using System;
using System.Collections.Generic;

namespace HypeDomino.UI.Cosmetics
{
    public interface ICosmeticArtSource<T> where T : class
    {
        void Load(string address, Action<T> done);

        void Release(string address);
    }

    public sealed class CosmeticArtCache<T> where T : class
    {
        sealed class Entry
        {
            public readonly List<Action<T>> Waiting = new();
            public int References;
            public bool Settled;
            public T Art;
        }

        readonly ICosmeticArtSource<T> source;
        readonly Dictionary<string, Entry> entries = new();

        public CosmeticArtCache(ICosmeticArtSource<T> source) => this.source = source;

        public void Acquire(string address, Action<T> ready)
        {
            if (entries.TryGetValue(address, out var known))
            {
                known.References++;
                if (known.Settled) ready(known.Art);
                else known.Waiting.Add(ready);
                return;
            }
            var entry = new Entry { References = 1 };
            entry.Waiting.Add(ready);
            entries[address] = entry;
            source.Load(address, art => Settle(address, entry, art));
        }

        public void Release(string address)
        {
            if (!entries.TryGetValue(address, out var entry)) return;
            entry.References--;
            if (entry.References == 0 && entry.Settled) Forget(address, entry);
        }

        void Settle(string address, Entry entry, T art)
        {
            entry.Art = art;
            for (var at = 0; at < entry.Waiting.Count; at++) entry.Waiting[at](art);
            entry.Waiting.Clear();
            entry.Settled = true;
            if (entry.References == 0) Forget(address, entry);
        }

        void Forget(string address, Entry entry)
        {
            entries.Remove(address);
            if (entry.Art != null) source.Release(address);
        }
    }
}
