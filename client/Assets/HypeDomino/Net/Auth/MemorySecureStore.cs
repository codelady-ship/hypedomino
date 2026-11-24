using System.Collections.Generic;

namespace HypeDomino.Net.Auth
{
    public sealed class MemorySecureStore : ISecureStore
    {
        readonly Dictionary<string, string> values = new();

        public string Read(string key) => values.TryGetValue(key, out var value) ? value : null;

        public void Write(string key, string value) => values[key] = value;

        public void Delete(string key) => values.Remove(key);
    }
}
