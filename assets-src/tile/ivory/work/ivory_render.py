import json
import math
import os
import sys

import bpy

WIDTH, LENGTH, THICKNESS = 0.025, 0.05, 0.0095
BEVEL = 0.0028
CANVAS = (128, 256)
SUPERSAMPLE = 4
SAMPLES = 64
SEED = 169
THREADS = 8
PIP_SPACING = 0.27
PIP_RADIUS = 0.088
DIVIDER_LENGTH = 0.74
DIVIDER_WIDTH = 0.0009
INLAY_LIFT = 0.00002
PIN_RADIUS = 0.075
PIN_FLATTEN = 0.55
LAYOUT = {
    0: [],
    1: [(0, 0)],
    2: [(-1, 1), (1, -1)],
    3: [(-1, 1), (0, 0), (1, -1)],
    4: [(-1, -1), (1, -1), (-1, 1), (1, 1)],
    5: [(-1, -1), (1, -1), (0, 0), (-1, 1), (1, 1)],
    6: [(-1, -1), (1, -1), (-1, 0), (1, 0), (-1, 1), (1, 1)],
}


def reset_scene():
    bpy.ops.wm.read_factory_settings(use_empty=True)
    scene = bpy.context.scene
    scene.render.engine = "CYCLES"
    scene.cycles.device = "CPU"
    scene.cycles.samples = SAMPLES
    scene.cycles.seed = SEED
    scene.cycles.use_animated_seed = False
    scene.cycles.use_adaptive_sampling = False
    scene.cycles.use_denoising = False
    scene.render.threads_mode = "FIXED"
    scene.render.threads = THREADS
    scene.render.film_transparent = True
    scene.render.resolution_x = CANVAS[0] * SUPERSAMPLE
    scene.render.resolution_y = CANVAS[1] * SUPERSAMPLE
    scene.render.resolution_percentage = 100
    scene.render.image_settings.file_format = "PNG"
    scene.render.image_settings.color_mode = "RGBA"
    scene.render.image_settings.color_depth = "8"
    scene.render.image_settings.compression = 90
    scene.view_settings.view_transform = "Standard"
    scene.view_settings.look = "None"
    scene.view_settings.exposure = 0.0
    scene.view_settings.gamma = 1.0
    scene.display_settings.display_device = "sRGB"
    for flag in [name for name in dir(scene.render) if name.startswith("use_stamp")]:
        setattr(scene.render, flag, False)
    scene.render.use_compositing = False
    scene.render.use_sequencer = False
    return scene


def material(name, spec, grain=False):
    result = bpy.data.materials.new(name)
    result.use_nodes = True
    nodes = result.node_tree
    shader = nodes.nodes["Principled BSDF"]
    shader.inputs["Base Color"].default_value = (*spec["color"], 1.0)
    shader.inputs["Roughness"].default_value = spec["roughness"]
    shader.inputs["Metallic"].default_value = spec.get("metallic", 0.0)
    shader.inputs["Specular IOR Level"].default_value = spec.get("specular", 0.5)
    shader.inputs["Coat Weight"].default_value = spec.get("coat", 0.0)
    shader.inputs["Coat Roughness"].default_value = spec.get("coatRoughness", 0.08)
    if spec.get("subsurface", 0.0):
        shader.inputs["Subsurface Weight"].default_value = spec["subsurface"]
        shader.inputs["Subsurface Radius"].default_value = (0.004, 0.003, 0.002)
    if grain:
        noise = nodes.nodes.new("ShaderNodeTexNoise")
        noise.inputs["Scale"].default_value = spec.get("grainScale", 900.0)
        bump = nodes.nodes.new("ShaderNodeBump")
        bump.inputs["Strength"].default_value = spec.get("grainStrength", 0.04)
        nodes.links.new(noise.outputs["Fac"], bump.inputs["Height"])
        nodes.links.new(bump.outputs["Normal"], shader.inputs["Normal"])
    return result


def select_only(target):
    bpy.ops.object.select_all(action="DESELECT")
    target.select_set(True)
    bpy.context.view_layer.objects.active = target


def tile_body(face):
    bpy.ops.mesh.primitive_cube_add(size=1)
    body = bpy.context.object
    body.scale = (WIDTH, LENGTH, THICKNESS)
    bpy.ops.object.transform_apply(scale=True)
    body.data.materials.append(face)
    bevel = body.modifiers.new("bevel", "BEVEL")
    bevel.width = BEVEL
    bevel.segments = 6
    bevel.limit_method = "NONE"
    bpy.ops.object.modifier_apply(modifier="bevel")
    return body


def pip_centres(top, bottom):
    spacing = WIDTH * PIP_SPACING
    for value, half_centre in ((top, LENGTH / 4), (bottom, -LENGTH / 4)):
        for column, row in LAYOUT[value]:
            yield column * spacing, half_centre + row * spacing


def inlays(top, bottom, ink, divider, brass):
    surface = THICKNESS / 2 + INLAY_LIFT
    for x, y in pip_centres(top, bottom):
        bpy.ops.mesh.primitive_cylinder_add(radius=WIDTH * PIP_RADIUS, depth=INLAY_LIFT, vertices=64, location=(x, y, surface))
        bpy.context.object.data.materials.append(ink)
    bpy.ops.mesh.primitive_cube_add(size=1, location=(0, 0, surface))
    line = bpy.context.object
    line.scale = (WIDTH * DIVIDER_LENGTH, DIVIDER_WIDTH, INLAY_LIFT)
    line.data.materials.append(divider)
    bpy.ops.mesh.primitive_uv_sphere_add(radius=WIDTH * PIN_RADIUS, location=(0, 0, THICKNESS / 2), segments=64, ring_count=32)
    pin = bpy.context.object
    pin.scale.z = PIN_FLATTEN
    pin.data.materials.append(brass)
    bpy.ops.object.shade_smooth()


def stage(scene, light):
    camera = bpy.data.objects.new("camera", bpy.data.cameras.new("camera"))
    camera.data.type = "ORTHO"
    camera.data.ortho_scale = LENGTH
    camera.location = (0, 0, 0.3)
    scene.collection.objects.link(camera)
    scene.camera = camera
    for name in ("key", "fill"):
        spec = light[name]
        lamp = bpy.data.objects.new(name, bpy.data.lights.new(name, "AREA"))
        lamp.data.energy = spec["energy"]
        lamp.data.size = spec["size"]
        lamp.data.color = spec.get("color", (1.0, 1.0, 1.0))
        lamp.location = spec["location"]
        lamp.rotation_euler = tuple(math.radians(angle) for angle in spec["rotation"])
        scene.collection.objects.link(lamp)
    world = bpy.data.worlds.new("world")
    world.use_nodes = True
    background = world.node_tree.nodes["Background"]
    background.inputs["Color"].default_value = (*light["world"]["color"], 1.0)
    background.inputs["Strength"].default_value = light["world"]["strength"]
    scene.world = world


def render_face(settings, top, bottom, out_dir):
    scene = reset_scene()
    look = settings["materials"]
    face = material("face", look["face"], grain=True)
    ink = material("pip", look["pip"])
    divider = material("divider", look["divider"])
    brass = material("pin", look["pin"])
    body = tile_body(face)
    select_only(body)
    bpy.ops.object.shade_smooth_by_angle(angle=math.radians(35))
    inlays(top, bottom, ink, divider, brass)
    stage(scene, settings["light"])
    scene.render.filepath = os.path.join(out_dir, f"{top}_{bottom}.png")
    bpy.ops.render.render(write_still=True)


def faces(selection):
    if selection:
        return [tuple(int(value) for value in item.split("_")) for item in selection]
    return [(low, high) for low in range(7) for high in range(low, 7)]


def main():
    arguments = sys.argv[sys.argv.index("--") + 1:]
    with open(arguments[0], encoding="utf-8") as source:
        settings = json.load(source)
    out_dir = os.path.abspath(arguments[1])
    os.makedirs(out_dir, exist_ok=True)
    for top, bottom in faces(arguments[2:]):
        render_face(settings, top, bottom, out_dir)
    print(f"rendered into {out_dir}")


main()
