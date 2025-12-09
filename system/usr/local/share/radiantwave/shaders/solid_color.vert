#version 330 core

layout (location = 0) in vec2 aPos;

uniform mat4 u_mvpMatrix;

void main() {
  gl_Position = u_mvpMatrix * vec4(aPos.xy, 0.0, 1.0);
}
