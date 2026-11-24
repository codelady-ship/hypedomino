using UnityEngine;

namespace HypeDomino.UI.Motion
{
    public readonly struct TransformTweenSink : ITweenSink
    {
        readonly Transform[] targets;

        public TransformTweenSink(Transform[] targets) => this.targets = targets;

        public void Apply(int target, TweenChannel channel, Vector3 value)
        {
            var moved = targets[target];
            switch (channel)
            {
                case TweenChannel.Position:
                    moved.localPosition = value;
                    return;
                case TweenChannel.Rotation:
                    moved.localRotation = Quaternion.Euler(value);
                    return;
                case TweenChannel.Scale:
                    moved.localScale = value;
                    return;
            }
        }
    }
}
