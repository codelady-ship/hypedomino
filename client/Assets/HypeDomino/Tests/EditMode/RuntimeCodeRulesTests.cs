using System;
using System.Collections.Generic;
using System.IO;
using System.Text.RegularExpressions;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class RuntimeCodeRulesTests
    {
        const string DesignTokensFile = "DesignTokens.cs";
        static readonly string[] NonRuntimeFolders = { "Editor", "Generated", "Tests" };
        static readonly string[] FileWriters = { "AddressablesCatalogSource.cs", "UnityStringBundleSource.cs", "UnityCosmeticManifestSource.cs" };
        static readonly Regex HandWrittenColour = new Regex(@"\bnew\s+Color(32)?\s*\(|\bColor\.(black|blue|cyan|gray|green|grey|magenta|red|white|yellow)\b");
        static readonly Regex PlayerPrefsKey = new Regex(@"\bPlayerPrefs\.(?:Get|Set|Has|Delete)\w*\(\s*(?:""(?<literal>[^""]*)""|(?<name>[A-Za-z_]\w*))");
        static readonly Regex FileWrite = new Regex(@"\bFile\.(?:Write|Append|Create|Copy|Move|Replace)\w*\s*\(|\bnew\s+(?:FileStream|StreamWriter|BinaryWriter)\s*\(");
        static readonly Regex CultureBlindCasing = new Regex(@"\.To(?:Upper|Lower)\(\s*\)|\b[Cc]har\.To(?:Upper|Lower)\(\s*[^,()]+\)|\bFontStyles\.(?:UpperCase|LowerCase|SmallCaps)\b");
        static readonly Regex DocumentedKey = new Regex(@"`([a-z]+(?:\.[A-Za-z]+)+)`");
        static readonly Regex KeyWord = new Regex(@"[A-Z]?[a-z]+|[A-Z]+(?![a-z])");
        static readonly HashSet<string> ServerValueWords = new HashSet<string>
        {
            "balance", "balances", "coin", "coins", "crystal", "crystals", "price", "prices", "cost", "costs",
            "amount", "amounts", "reward", "rewards", "score", "scores", "points", "rating", "ratings", "mmr",
            "lp", "xp", "rank", "tier", "inventory", "owned", "result", "results",
        };

        [Test]
        public void RuntimeCodeDoesNotUseLinq()
        {
            var offenders = new List<string>();
            foreach (var file in RuntimeSources())
                if (File.ReadAllText(file).Contains("System.Linq")) offenders.Add(file);
            Assert.IsEmpty(offenders);
        }

        [Test]
        public void RuntimeCodeTakesColoursFromDesignTokens()
        {
            var offenders = new List<string>();
            foreach (var file in RuntimeSources())
                if (Path.GetFileName(file) != DesignTokensFile && HandWrittenColour.IsMatch(File.ReadAllText(file))) offenders.Add(file);
            Assert.IsEmpty(offenders);
        }

        [TestCase("image.color = new Color(1f, 0f, 0f);")]
        [TestCase("var tint = new Color32(255, 0, 0, 255);")]
        [TestCase("label.color = Color.white;")]
        public void HandWrittenColoursAreRecognised(string line) => Assert.IsTrue(HandWrittenColour.IsMatch(line));

        [TestCase("image.color = TokenColor.RolePrimaryFill;")]
        [TestCase("camera.backgroundColor = Color.clear;")]
        [TestCase("var colourCount = palette.Length;")]
        public void TokenColoursAreNotFlagged(string line) => Assert.IsFalse(HandWrittenColour.IsMatch(line));

        [Test]
        public void RuntimeCodeDoesNotChangeCaseWithoutACulture()
        {
            var offenders = new List<string>();
            foreach (var file in RuntimeSources())
                if (CultureBlindCasing.IsMatch(File.ReadAllText(file))) offenders.Add(file);
            Assert.IsEmpty(offenders, "docs/localization.md 4.6: display text is cased with the selected language's culture, identifiers with the invariant one");
        }

        [TestCase("var title = text.ToUpper();")]
        [TestCase("label.text = name.ToLower( );")]
        [TestCase("var first = char.ToUpper(name[0]);")]
        [TestCase("var first = Char.ToLower(letter);")]
        [TestCase("label.fontStyle = FontStyles.UpperCase;")]
        [TestCase("label.fontStyle |= FontStyles.SmallCaps;")]
        public void CultureBlindCasingIsRecognised(string line) => Assert.IsTrue(CultureBlindCasing.IsMatch(line));

        [TestCase("var code = text.ToUpperInvariant();")]
        [TestCase("var tag = locale.ToLowerInvariant();")]
        [TestCase("var title = text.ToUpper(culture);")]
        [TestCase("var upper = char.ToUpperInvariant(character);")]
        [TestCase("var first = char.ToUpper(name[0], culture);")]
        [TestCase("label.fontStyle = FontStyles.Bold;")]
        public void CulturedOrInvariantCasingIsNotFlagged(string line) => Assert.IsFalse(CultureBlindCasing.IsMatch(line));

        [Test]
        public void EveryPlayerPrefsKeyIsListedUnderDeviceStorage()
        {
            var documented = DeviceStorageKeys();
            var used = UsedPlayerPrefsKeys();
            CollectionAssert.IsSubsetOf(used, documented, "a PlayerPrefs key used in runtime code is missing from docs/anti-cheat.md, Device storage");
            CollectionAssert.IsSubsetOf(documented, used, "docs/anti-cheat.md, Device storage lists a key runtime code no longer uses");
        }

        [Test]
        public void NoDeviceStorageKeyNamesAServerValue()
        {
            var offenders = new List<string>();
            foreach (var key in DeviceStorageKeys())
                if (NamesServerValue(key)) offenders.Add(key);
            foreach (var key in UsedPlayerPrefsKeys())
                if (NamesServerValue(key)) offenders.Add(key);
            Assert.IsEmpty(offenders);
        }

        [Test]
        public void OnlyTheListedRuntimeFilesWriteFiles()
        {
            var offenders = new List<string>();
            foreach (var file in RuntimeSources())
                if (Array.IndexOf(FileWriters, Path.GetFileName(file)) < 0 && FileWrite.IsMatch(File.ReadAllText(file))) offenders.Add(file);
            Assert.IsEmpty(offenders);
        }

        [TestCase("wallet.coinBalance")]
        [TestCase("shop.lastPrice")]
        [TestCase("profile.XP")]
        [TestCase("match.score")]
        public void ServerValueKeysAreRecognised(string key) => Assert.IsTrue(NamesServerValue(key));

        [TestCase("graphics.choice")]
        [TestCase("content.catalogVersion")]
        [TestCase("audio.helpShown")]
        public void SettingKeysAreNotFlagged(string key) => Assert.IsFalse(NamesServerValue(key));

        [TestCase("PlayerPrefs.SetInt(ChoiceKey, 1);", "ChoiceKey")]
        [TestCase("PlayerPrefs.GetString( \"content.baseUrl\", null);", "content.baseUrl")]
        [TestCase("PlayerPrefs.HasKey(FrameRateKey)", "FrameRateKey")]
        public void PlayerPrefsCallsAreRecognised(string line, string key)
        {
            var match = PlayerPrefsKey.Match(line);
            Assert.IsTrue(match.Success);
            Assert.AreEqual(key, match.Groups["literal"].Success ? match.Groups["literal"].Value : match.Groups["name"].Value);
        }

        [TestCase("File.WriteAllText(path, text);")]
        [TestCase("using var stream = new FileStream(path, FileMode.Create);")]
        [TestCase("File.AppendAllText(path, line);")]
        public void FileWritesAreRecognised(string line) => Assert.IsTrue(FileWrite.IsMatch(line));

        static bool NamesServerValue(string key)
        {
            foreach (Match word in KeyWord.Matches(key))
                if (ServerValueWords.Contains(word.Value.ToLowerInvariant())) return true;
            return false;
        }

        static HashSet<string> DeviceStorageKeys()
        {
            var path = Path.GetFullPath(Path.Combine(Application.dataPath, "..", "..", "docs", "anti-cheat.md"));
            var keys = new HashSet<string>();
            var inSection = false;
            foreach (var line in File.ReadAllLines(path))
            {
                if (line.StartsWith("#")) inSection = line.StartsWith("### Device storage");
                if (!inSection || !line.StartsWith("| `")) continue;
                var firstCell = line.Split('|')[1];
                foreach (Match key in DocumentedKey.Matches(firstCell)) keys.Add(key.Groups[1].Value);
            }
            Assert.IsNotEmpty(keys, "docs/anti-cheat.md has no Device storage table");
            return keys;
        }

        [Test]
        public void TheFileWritersAreNamedUnderDeviceStorage()
        {
            var doc = File.ReadAllText(Path.GetFullPath(Path.Combine(Application.dataPath, "..", "..", "docs", "anti-cheat.md")));
            foreach (var writer in FileWriters) StringAssert.Contains(writer, doc);
        }

        static HashSet<string> UsedPlayerPrefsKeys()
        {
            var keys = new HashSet<string>();
            foreach (var file in RuntimeSources())
            {
                var source = File.ReadAllText(file);
                foreach (Match call in PlayerPrefsKey.Matches(source))
                    keys.Add(call.Groups["literal"].Success ? call.Groups["literal"].Value : ConstantValue(source, call.Groups["name"].Value, file));
            }
            return keys;
        }

        static string ConstantValue(string source, string name, string file)
        {
            var constant = Regex.Match(source, @"\bconst\s+string\s+" + Regex.Escape(name) + @"\s*=\s*""([^""]*)""");
            Assert.IsTrue(constant.Success, $"{file}: PlayerPrefs key {name} is not a string constant in the same file");
            return constant.Groups[1].Value;
        }

        internal static IEnumerable<string> RuntimeSources()
        {
            var root = Path.Combine(Application.dataPath, "HypeDomino");
            foreach (var folder in Directory.GetDirectories(root))
            {
                if (Array.IndexOf(NonRuntimeFolders, Path.GetFileName(folder)) >= 0) continue;
                foreach (var file in Directory.GetFiles(folder, "*.cs", SearchOption.AllDirectories))
                    yield return file;
            }
        }
    }
}
