import argparse
import os
import re
import sys

import bpy

FRAME_NAME = re.compile(r"^\d{4}\.png$")


def parse_arguments(argv):
    arguments = argv[argv.index("--") + 1:] if "--" in argv else []
    parser = argparse.ArgumentParser(prog="export_loop.py")
    parser.add_argument("--frames", help="folder for the loop: export/anim or export/layer/<layer>")
    parser.add_argument("--static", help="file for the static fallback: export/static.png")
    parser.add_argument("--static-frame", type=int, help="scene frame chosen as the static fallback")
    parsed = parser.parse_args(arguments)
    if not parsed.frames and not parsed.static:
        parser.error("give --frames, --static or both")
    if parsed.static and parsed.static_frame is None:
        parser.error("--static needs --static-frame, a chosen key frame")
    return parsed


def scene_error(scene, arguments):
    if scene.camera is None:
        return "the scene has no camera"
    if scene.camera.data.type != "ORTHO":
        return "the camera must be orthographic (docs/asset-workflow.md 5.2)"
    if arguments.static and not scene.frame_start <= arguments.static_frame <= scene.frame_end:
        return f"--static-frame {arguments.static_frame} is outside the scene range {scene.frame_start} to {scene.frame_end}"
    return None


def apply_output_settings(scene):
    render = scene.render
    render.film_transparent = True
    render.resolution_percentage = 100
    render.use_compositing = False
    render.use_sequencer = False
    render.use_file_extension = True
    image = render.image_settings
    image.file_format = "PNG"
    image.color_mode = "RGBA"
    image.color_depth = "8"
    scene.display_settings.display_device = "sRGB"
    scene.view_settings.view_transform = "Standard"
    scene.view_settings.look = "None"
    scene.view_settings.exposure = 0.0
    scene.view_settings.gamma = 1.0


def render_frame(scene, frame, path):
    scene.frame_set(frame)
    scene.render.filepath = path
    bpy.ops.render.render(write_still=True)


def clear_old_frames(folder):
    os.makedirs(folder, exist_ok=True)
    for name in os.listdir(folder):
        if FRAME_NAME.match(name):
            os.remove(os.path.join(folder, name))


def export(scene, arguments):
    written = []
    if arguments.frames:
        clear_old_frames(arguments.frames)
        frames = range(scene.frame_start, scene.frame_end + 1, scene.frame_step)
        for index, frame in enumerate(frames, start=1):
            path = os.path.join(arguments.frames, f"{index:04d}.png")
            render_frame(scene, frame, path)
            written.append(path)
    if arguments.static:
        os.makedirs(os.path.dirname(os.path.abspath(arguments.static)), exist_ok=True)
        render_frame(scene, arguments.static_frame, arguments.static)
        written.append(arguments.static)
    return written


def main():
    arguments = parse_arguments(sys.argv)
    scene = bpy.context.scene
    error = scene_error(scene, arguments)
    if error:
        print(f"export_loop: {error}", file=sys.stderr)
        sys.exit(1)
    apply_output_settings(scene)
    written = export(scene, arguments)
    print(f"export_loop: wrote {len(written)} files")


main()
