package game

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os/exec"
	"slices"
	"time"

	"github.com/mirstar13/go-3d-graphics/internal/engine"
	"github.com/mirstar13/go-3d-graphics/internal/physics"
	"github.com/mirstar13/go-3d-graphics/internal/project"
)

const behaviorProtocol = 1
const behaviorTimeout = time.Second
const behaviorMessageLimit = 256 << 10

type behaviorActor struct {
	ID         string          `json:"id"`
	Behavior   string          `json:"behavior"`
	Parameters json.RawMessage `json:"parameters"`
	Position   [3]float64      `json:"position"`
	Velocity   [3]float64      `json:"velocity"`
}

type behaviorContact struct {
	A      string `json:"a"`
	B      string `json:"b"`
	Phase  string `json:"phase"`
	Sensor bool   `json:"sensor"`
}

type behaviorRequest struct {
	Version   int               `json:"version"`
	Tick      uint64            `json:"tick"`
	Delta     float64           `json:"delta"`
	Start     bool              `json:"start"`
	Actions   []string          `json:"actions"`
	Actors    []behaviorActor   `json:"actors"`
	SoundCues []string          `json:"sound_cues,omitempty"`
	Contacts  []behaviorContact `json:"contacts"`
}

type behaviorReply struct {
	Version  int               `json:"version"`
	Tick     uint64            `json:"tick"`
	Impulses []behaviorImpulse `json:"impulses"`
	Sounds   []string          `json:"sounds,omitempty"`
	Error    string            `json:"error,omitempty"`
}

type behaviorImpulse struct {
	Entity string    `json:"entity"`
	Vector []float64 `json:"vector"`
}

type behaviorStderr struct{ buffer bytes.Buffer }

func (b *behaviorStderr) Write(data []byte) (int, error) {
	n := min(len(data), 2048-b.buffer.Len())
	_, _ = b.buffer.Write(data[:n])
	return len(data), nil
}

type behaviorWorker struct {
	ctx     context.Context
	project *project.Project
	command *exec.Cmd
	out     io.ReadCloser
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	actions []string
	failure error
	stderr  behaviorStderr
}

// EnableBehaviors opts this runtime into external project code. The worker
// starts on the first fixed tick with an attached behavior.
func (s *Session) EnableBehaviors(ctx context.Context, p *project.Project) {
	if p.Manifest.BehaviorWorker != "" && s.behaviorWorker == nil {
		s.behaviorWorker = &behaviorWorker{ctx: ctx, project: p}
	}
}

func (s *Session) BehaviorInput(keys ...string) {
	w := s.behaviorWorker
	if w == nil {
		return
	}
	for _, key := range keys {
		action := w.project.Manifest.GameplayBindings[key]
		if action != "" && len(w.actions) < 64 && !slices.Contains(w.actions, action) {
			w.actions = append(w.actions, action)
		}
	}
}

// CloseBehaviors releases any external worker owned by this runtime.
func (s *Session) CloseBehaviors() {
	if w := s.behaviorWorker; w != nil {
		w.shutdown()
		s.behaviorWorker = nil
	}
}

func (w *behaviorWorker) start() error {
	file, err := w.project.WorkerExecutable()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(w.ctx, file)
	cmd.Dir = w.project.Root
	cmd.WaitDelay = 150 * time.Millisecond
	cmd.Stderr = &w.stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return err
	}
	if err = cmd.Start(); err != nil {
		in.Close()
		out.Close()
		return err
	}
	w.command, w.stdin, w.out = cmd, in, out
	w.scanner = bufio.NewScanner(out)
	w.scanner.Buffer(make([]byte, 4096), behaviorMessageLimit)
	return nil
}

func (w *behaviorWorker) close() {
	if w.command == nil {
		return
	}
	_ = w.command.Process.Kill()
	_ = w.stdin.Close()
	_ = w.out.Close()
	_ = w.command.Wait()
	w.command = nil
}

func (w *behaviorWorker) shutdown() {
	if w.command == nil {
		return
	}
	_ = w.stdin.Close()
	cmd := w.command
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		_ = cmd.Process.Kill()
		<-done
	}
	w.command = nil
}

func (w *behaviorWorker) exchange(request behaviorRequest) (behaviorReply, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return behaviorReply{}, err
	}
	if len(data) >= behaviorMessageLimit {
		return behaviorReply{}, fmt.Errorf("behavior request exceeds message budget")
	}
	type result struct {
		reply behaviorReply
		err   error
	}
	done := make(chan result, 1)
	stdin, scanner := w.stdin, w.scanner
	go func() {
		var r result
		if _, r.err = stdin.Write(append(data, '\n')); r.err != nil {
			done <- r
			return
		}
		if !scanner.Scan() {
			r.err = scanner.Err()
			if r.err == nil {
				r.err = io.ErrUnexpectedEOF
			}
		} else {
			dec := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
			dec.DisallowUnknownFields()
			r.err = dec.Decode(&r.reply)
			if r.err == nil && dec.Decode(&struct{}{}) != io.EOF {
				r.err = fmt.Errorf("expected one JSON reply")
			}
		}
		done <- r
	}()
	timer := time.NewTimer(behaviorTimeout)
	defer timer.Stop()
	select {
	case r := <-done:
		return r.reply, r.err
	case <-w.ctx.Done():
		return behaviorReply{}, w.ctx.Err()
	case <-timer.C:
		return behaviorReply{}, fmt.Errorf("worker did not reply within %s", behaviorTimeout)
	}
}

func (s *Session) runBehaviors(t physics.Tick) error {
	w := s.behaviorWorker
	if w == nil {
		return nil
	}
	if w.failure != nil {
		return w.failure
	}
	r := behaviorRequest{Version: behaviorProtocol, Tick: t.Index, Delta: t.Delta, Start: w.command == nil,
		Actions: append([]string{}, w.actions...), Actors: []behaviorActor{}, Contacts: []behaviorContact{}}
	w.actions = w.actions[:0]
	r.SoundCues = s.soundCues
	ids := make(map[physics.BodyID]string)
	for i := range s.entities {
		e := &s.entities[i]
		if !e.active {
			continue
		}
		description := e.description
		if description.Body != "none" {
			ids[e.body] = description.ID
		}
		if description.Behavior == "" {
			continue
		}
		params := json.RawMessage(description.BehaviorArgs)
		if len(params) == 0 {
			params = json.RawMessage("{}")
		}
		p := e.node.Transform.GetWorldPosition()
		actor := behaviorActor{ID: description.ID, Behavior: description.Behavior, Parameters: params, Position: [3]float64{p.X, p.Y, p.Z}}
		if description.Body != "none" {
			state, err := s.World.Body(e.body)
			if err == nil {
				v := state.Velocity
				actor.Velocity = [3]float64{v.X, v.Y, v.Z}
			}
		}
		r.Actors = append(r.Actors, actor)
	}
	if len(r.Actors) == 0 {
		return nil
	}
	for _, event := range s.World.ContactEvents() {
		a, b := ids[event.A], ids[event.B]
		if a == "" || b == "" {
			continue
		}
		phase := []string{"begin", "stay", "end"}[event.Phase]
		r.Contacts = append(r.Contacts, behaviorContact{a, b, phase, event.Sensor})
	}
	slices.SortFunc(r.Actors, func(a, b behaviorActor) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	err := s.requestBehaviors(w, r)
	if err != nil {
		w.close()
		w.failure = fmt.Errorf("behavior tick %d: %w", t.Index, err)
		if w.stderr.buffer.Len() > 0 {
			w.failure = fmt.Errorf("%w (worker stderr: %q)", w.failure, w.stderr.buffer.String())
		}
	}
	return w.failure
}

func (s *Session) requestBehaviors(w *behaviorWorker, r behaviorRequest) error {
	if w.command == nil {
		if err := w.start(); err != nil {
			return err
		}
	}
	reply, err := w.exchange(r)
	if err != nil {
		return err
	}
	if reply.Version != behaviorProtocol || reply.Tick != r.Tick || len(reply.Impulses) > 64-len(s.pending) {
		return fmt.Errorf("invalid worker version, tick or impulse count")
	}
	if reply.Error != "" {
		if !project.ValidText(reply.Error, 256) {
			return fmt.Errorf("worker returned an invalid error message")
		}
		return fmt.Errorf("worker: %s", reply.Error)
	}
	if len(reply.Sounds) > maxTickSounds-len(s.pendingSounds) {
		return fmt.Errorf("too many worker sounds")
	}
	for _, cue := range reply.Sounds {
		if s.sounds[cue] == nil {
			return fmt.Errorf("unknown worker audio cue %q", cue)
		}
	}
	handles := make([]EntityID, len(reply.Impulses))
	for i, command := range reply.Impulses {
		id, ok := s.Find(command.Entity)
		if !ok || s.entities[id.slot].description.Body != "dynamic" || len(command.Vector) != 3 {
			return fmt.Errorf("invalid worker impulse for %q", command.Entity)
		}
		for _, component := range command.Vector {
			if math.IsNaN(component) || math.Abs(component) > 1e6 {
				return fmt.Errorf("invalid worker impulse component")
			}
		}
		handles[i] = id
	}
	initial := len(s.pending)
	for i, command := range reply.Impulses {
		v := command.Vector
		if err := s.QueueImpulse(handles[i], engine.Point{X: v[0], Y: v[1], Z: v[2]}); err != nil {
			s.pending = s.pending[:initial]
			return err
		}
	}
	for _, cue := range reply.Sounds {
		s.pendingSounds = append(s.pendingSounds, s.sounds[cue])
	}
	return nil
}
