using System.Text.Json;

while (Console.ReadLine() is { } line)
{
    using var document = JsonDocument.Parse(line);
    var request = document.RootElement;
    if (request.GetProperty("version").GetInt32() != 1)
        throw new InvalidDataException("Unsupported behavior protocol");

    bool launched = request.GetProperty("actions").EnumerateArray()
        .Any(action => action.GetString() == "launch");
    var impulses = new List<Impulse>();
    foreach (var actor in request.GetProperty("actors").EnumerateArray())
    {
        if (actor.GetProperty("behavior").GetString() != "launch")
            throw new InvalidDataException("Unknown behavior");
        if (!launched)
            continue;
        var parameters = actor.GetProperty("parameters");
        double strength = parameters.TryGetProperty("height_impulse", out var value)
            ? value.GetDouble() : 5.0;
        string entity = actor.GetProperty("id").GetString() ?? throw new InvalidDataException("Invalid actor ID");
        impulses.Add(new Impulse(entity, [0, strength, 0]));
    }
    var reply = new Dictionary<string, object>
    {
        ["version"] = 1,
        ["tick"] = request.GetProperty("tick").GetUInt64(),
        ["impulses"] = impulses,
    };
    if (launched && request.TryGetProperty("sound_cues", out var cues)
        && cues.EnumerateArray().Any(cue => cue.GetString() == "launch"))
        reply["sounds"] = new[] { "launch" };
    Console.WriteLine(JsonSerializer.Serialize(reply, new JsonSerializerOptions { PropertyNamingPolicy = JsonNamingPolicy.CamelCase }));
}

record Impulse(string Entity, double[] Vector);
