using System;
using System.Collections.Generic;
using HypeDomino.UI.Cosmetics;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class FakeArtSource<T> : ICosmeticArtSource<T> where T : class
    {
        readonly Dictionary<string, Action<T>> pending = new();

        public readonly List<string> Loads = new();
        public readonly List<string> Releases = new();

        public void Load(string address, Action<T> done)
        {
            Loads.Add(address);
            pending[address] = done;
        }

        public void Release(string address) => Releases.Add(address);

        public void Finish(string address, T art)
        {
            var done = pending[address];
            pending.Remove(address);
            done(art);
        }
    }
}
