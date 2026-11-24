import bpy

scene = bpy.context.scene
scene.render.engine = "BLENDER_WORKBENCH"
scene.render.resolution_x = 32
scene.render.resolution_y = 32
scene.render.resolution_percentage = 50
scene.frame_start = 1
scene.frame_end = 3
scene.view_settings.view_transform = "AgX"
scene.render.film_transparent = False
scene.camera.data.type = "ORTHO"
cube = bpy.data.objects["Cube"]
cube.rotation_euler = (0.0, 0.0, 0.0)
cube.keyframe_insert("rotation_euler", frame=1)
cube.rotation_euler = (0.0, 0.0, 1.0)
cube.keyframe_insert("rotation_euler", frame=3)
