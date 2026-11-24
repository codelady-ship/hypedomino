using System.Collections.Generic;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class MeshProbe : MonoBehaviour, IMeshModifier
    {
        public readonly List<UIVertex> Vertices = new();

        public void ModifyMesh(Mesh mesh) { }

        public void ModifyMesh(VertexHelper verts)
        {
            Vertices.Clear();
            verts.GetUIVertexStream(Vertices);
        }
    }
}
