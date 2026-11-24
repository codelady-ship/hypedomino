using System;
using System.Collections.Generic;
using System.IO;
using System.IO.Compression;
using System.Text;

namespace HypeDomino.Editor
{
    public static class UnityPackageFiles
    {
        const int BlockSize = 512;
        const int NameLength = 100;
        const int SizeOffset = 124;
        const int SizeLength = 12;
        const int TypeOffset = 156;
        const int PrefixOffset = 345;
        const int PrefixLength = 155;
        const int OctalBase = 8;

        public static int Extract(string package, Func<string, bool> wanted)
        {
            var written = 0;
            foreach (var entry in Entries(package).Values)
            {
                if (!entry.TryGetValue("pathname", out var pathname)) continue;
                var path = Encoding.UTF8.GetString(pathname).Split('\n')[0].Trim();
                if (!wanted(path)) continue;
                if (entry.TryGetValue("asset", out var asset))
                {
                    Directory.CreateDirectory(Path.GetDirectoryName(path));
                    File.WriteAllBytes(path, asset);
                }
                else Directory.CreateDirectory(path);
                if (entry.TryGetValue("asset.meta", out var meta)) File.WriteAllBytes(path + ".meta", meta);
                written++;
            }
            return written;
        }

        static Dictionary<string, Dictionary<string, byte[]>> Entries(string package)
        {
            var entries = new Dictionary<string, Dictionary<string, byte[]>>();
            using var stream = new GZipStream(File.OpenRead(package), CompressionMode.Decompress);
            var header = new byte[BlockSize];
            while (Fill(stream, header) && header[0] != 0)
            {
                var size = Convert.ToInt64(Field(header, SizeOffset, SizeLength).Trim(), OctalBase);
                var data = new byte[size];
                if (!Fill(stream, data)) throw new EndOfStreamException(package);
                Skip(stream, (BlockSize - size % BlockSize) % BlockSize);
                var type = (char)header[TypeOffset];
                if (type != '0' && type != '\0') continue;
                var prefix = Field(header, PrefixOffset, PrefixLength);
                var name = Field(header, 0, NameLength);
                var parts = (prefix.Length > 0 ? prefix + "/" + name : name).Replace("./", string.Empty).Split('/');
                if (parts.Length != 2) continue;
                if (!entries.TryGetValue(parts[0], out var files)) entries[parts[0]] = files = new Dictionary<string, byte[]>();
                files[parts[1]] = data;
            }
            return entries;
        }

        static string Field(byte[] header, int offset, int length) => Encoding.ASCII.GetString(header, offset, length).Split('\0')[0];

        static bool Fill(Stream stream, byte[] buffer)
        {
            var read = 0;
            while (read < buffer.Length)
            {
                var count = stream.Read(buffer, read, buffer.Length - read);
                if (count == 0) return false;
                read += count;
            }
            return true;
        }

        static void Skip(Stream stream, long count)
        {
            if (count > 0) Fill(stream, new byte[count]);
        }
    }
}
