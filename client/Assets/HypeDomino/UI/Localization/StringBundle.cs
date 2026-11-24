using System.Collections.Generic;
using System.Globalization;
using System.Security.Cryptography;
using System.Text;

namespace HypeDomino.UI.Localization
{
    public sealed class StringBundle
    {
        readonly Dictionary<string, string> strings;

        StringBundle(string language, int version, string hash, Dictionary<string, string> strings)
        {
            Language = language;
            Version = version;
            Hash = hash;
            this.strings = strings;
        }

        public string Language { get; }

        public int Version { get; }

        public string Hash { get; }

        public bool TryGet(string key, out string value) => strings.TryGetValue(key, out value);

        public static bool TryRead(byte[] content, int version, string hash, out StringBundle bundle)
        {
            bundle = null;
            if (content == null || Sha256Hex(content) != hash) return false;
            var reader = new Reader(Encoding.UTF8.GetString(content));
            if (!reader.Bundle(out var language, out var strings)) return false;
            bundle = new StringBundle(language, version, hash, strings);
            return true;
        }

        public static string Sha256Hex(byte[] content)
        {
            using var sha = SHA256.Create();
            var sum = sha.ComputeHash(content);
            var hex = new StringBuilder(sum.Length * 2);
            for (var i = 0; i < sum.Length; i++) hex.Append(sum[i].ToString("x2", CultureInfo.InvariantCulture));
            return hex.ToString();
        }

        sealed class Reader
        {
            const string LanguageField = "language";
            const string StringsField = "strings";
            const int UnicodeEscapeLength = 4;

            readonly string json;
            int pos;

            public Reader(string json) => this.json = json;

            public bool Bundle(out string language, out Dictionary<string, string> strings)
            {
                language = null;
                strings = null;
                if (!Take('{')) return false;
                do
                {
                    if (!ReadString(out var field) || !Take(':')) return false;
                    var ok = field switch
                    {
                        LanguageField => ReadString(out language),
                        StringsField => ReadMap(out strings),
                        _ => false,
                    };
                    if (!ok) return false;
                }
                while (Take(','));
                return Take('}') && AtEnd() && language != null && strings != null;
            }

            bool ReadMap(out Dictionary<string, string> map)
            {
                map = new Dictionary<string, string>();
                if (!Take('{')) return false;
                if (Take('}')) return true;
                do
                {
                    if (!ReadString(out var key) || !Take(':') || !ReadString(out var value)) return false;
                    map[key] = value;
                }
                while (Take(','));
                return Take('}');
            }

            bool ReadString(out string value)
            {
                value = null;
                if (!Take('"')) return false;
                var builder = new StringBuilder();
                while (pos < json.Length)
                {
                    var c = json[pos++];
                    if (c == '"')
                    {
                        value = builder.ToString();
                        return true;
                    }
                    if (c != '\\')
                    {
                        builder.Append(c);
                        continue;
                    }
                    if (pos >= json.Length || !AppendEscape(builder, json[pos++])) return false;
                }
                return false;
            }

            bool AppendEscape(StringBuilder builder, char escape)
            {
                switch (escape)
                {
                    case '"':
                    case '\\':
                    case '/':
                        builder.Append(escape);
                        return true;
                    case 'b':
                        builder.Append('\b');
                        return true;
                    case 'f':
                        builder.Append('\f');
                        return true;
                    case 'n':
                        builder.Append('\n');
                        return true;
                    case 'r':
                        builder.Append('\r');
                        return true;
                    case 't':
                        builder.Append('\t');
                        return true;
                    case 'u':
                        return AppendUnicode(builder);
                    default:
                        return false;
                }
            }

            bool AppendUnicode(StringBuilder builder)
            {
                if (pos + UnicodeEscapeLength > json.Length) return false;
                if (!ushort.TryParse(json.Substring(pos, UnicodeEscapeLength), NumberStyles.AllowHexSpecifier, CultureInfo.InvariantCulture, out var code)) return false;
                builder.Append((char)code);
                pos += UnicodeEscapeLength;
                return true;
            }

            bool Take(char c)
            {
                SkipSpace();
                if (pos >= json.Length || json[pos] != c) return false;
                pos++;
                return true;
            }

            bool AtEnd()
            {
                SkipSpace();
                return pos == json.Length;
            }

            void SkipSpace()
            {
                while (pos < json.Length && char.IsWhiteSpace(json[pos])) pos++;
            }
        }
    }
}
