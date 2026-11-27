#version 330 core

uniform usampler2D colors;
layout(location = 0) out uvec4 pixel;

ivec3 rgb(ivec2 cell) {
    return ivec3(texelFetch(colors, cell, 0).rgb);
}

int luma(ivec3 c) {
    return 299 * c.r + 587 * c.g + 114 * c.b;
}

void main() {
    ivec2 at = ivec2(gl_FragCoord.xy);
    ivec2 size = textureSize(colors, 0);
    ivec3 center = rgb(at);
    pixel = uvec4(center, 255);
    if (any(equal(at, ivec2(0))) || any(equal(at, size - 1))) return;

    ivec3 north = rgb(at + ivec2(0, -1)), south = rgb(at + ivec2(0, 1));
    ivec3 east = rgb(at + ivec2(1, 0)), west = rgb(at + ivec2(-1, 0));
    int c = luma(center), n = luma(north), s = luma(south), e = luma(east), w = luma(west);
    int high = max(c, max(max(n, s), max(e, w)));
    int low = min(c, min(min(n, s), min(e, w)));
    int contrast = high - low;
    if (contrast <= 31 || 8 * contrast < high) return;

    int nw = luma(rgb(at + ivec2(-1, -1))), ne = luma(rgb(at + ivec2(1, -1)));
    int sw = luma(rgb(at + ivec2(-1, 1))), se = luma(rgb(at + ivec2(1, 1)));
    int corners = nw + ne + sw + se;
    bool horizontal = abs(corners - 2 * (w + c + e)) >= abs(corners - 2 * (n + c + s));
    if (horizontal && abs((nw + ne - 2 * n) - (sw + se - 2 * s)) > contrast) {
        pixel = uvec4((center + east + west) / 3, 255);
    } else if (!horizontal && abs((nw + sw - 2 * w) - (ne + se - 2 * e)) > contrast) {
        pixel = uvec4((center + north + south) / 3, 255);
    }
}
