using System;
using System.Collections.Generic;
using System.IO;
using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;

namespace HypeDomino.Net
{
    public sealed class JsonMap
    {
        const double LargestExactInteger = 9_007_199_254_740_992d;

        readonly Struct members;

        JsonMap(Struct members) => this.members = members;

        public IEnumerable<string> Keys => members.Fields.Keys;

        public static bool TryParse(string json, out JsonMap map)
        {
            map = null;
            if (string.IsNullOrEmpty(json)) return false;
            try
            {
                map = new JsonMap(JsonParser.Default.Parse<Struct>(json));
                return true;
            }
            catch (IOException)
            {
                return false;
            }
        }

        public bool IsAbsent(string key) => !members.Fields.TryGetValue(key, out var value) || value.KindCase == Value.KindOneofCase.NullValue;

        public bool TryMap(string key, out JsonMap map)
        {
            map = null;
            if (!TryValue(key, Value.KindOneofCase.StructValue, out var value)) return false;
            map = new JsonMap(value.StructValue);
            return true;
        }

        public bool TryString(string key, out string text)
        {
            text = null;
            if (!TryValue(key, Value.KindOneofCase.StringValue, out var value)) return false;
            text = value.StringValue;
            return true;
        }

        public bool TryInteger(string key, out long number)
        {
            number = 0;
            if (!TryValue(key, Value.KindOneofCase.NumberValue, out var value)) return false;
            var raw = value.NumberValue;
            if (Math.Floor(raw) != raw || Math.Abs(raw) > LargestExactInteger) return false;
            number = (long)raw;
            return true;
        }

        public bool TryStrings(string key, out string[] texts)
        {
            texts = null;
            if (!TryValue(key, Value.KindOneofCase.ListValue, out var value)) return false;
            var items = value.ListValue.Values;
            var read = new string[items.Count];
            for (var i = 0; i < read.Length; i++)
            {
                if (items[i].KindCase != Value.KindOneofCase.StringValue) return false;
                read[i] = items[i].StringValue;
            }
            texts = read;
            return true;
        }

        public bool TryMaps(string key, out JsonMap[] maps)
        {
            maps = null;
            if (!TryValue(key, Value.KindOneofCase.ListValue, out var value)) return false;
            var items = value.ListValue.Values;
            var read = new JsonMap[items.Count];
            for (var i = 0; i < read.Length; i++)
            {
                if (items[i].KindCase != Value.KindOneofCase.StructValue) return false;
                read[i] = new JsonMap(items[i].StructValue);
            }
            maps = read;
            return true;
        }

        bool TryValue(string key, Value.KindOneofCase kind, out Value value) =>
            members.Fields.TryGetValue(key, out value) && value.KindCase == kind;
    }
}
