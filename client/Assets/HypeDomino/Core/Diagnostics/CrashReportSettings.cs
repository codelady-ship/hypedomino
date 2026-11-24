using System;
using System.Text.RegularExpressions;
using UnityEngine;

namespace HypeDomino.Core.Diagnostics
{
    [Serializable]
    public sealed class CrashReportSettings
    {
        public const string ResourceName = "CrashReporting";
        static readonly Regex EnvironmentPattern = new Regex(@"^[A-Za-z0-9._-]{1,64}\z", RegexOptions.CultureInvariant);

        public string dsn;
        public string environment;

        public SentryDsn Dsn => SentryDsn.Parse(dsn);

        public static CrashReportSettings Load()
        {
            var asset = Resources.Load<TextAsset>(ResourceName);
            return asset == null ? null : FromJson(asset.text);
        }

        public static CrashReportSettings FromJson(string json)
        {
            var read = JsonUtility.FromJson<CrashReportSettings>(json) ?? throw new ArgumentException("the crash reporting settings are empty");
            if (string.IsNullOrEmpty(read.dsn)) return null;
            SentryDsn.Parse(read.dsn);
            if (!string.IsNullOrEmpty(read.environment) && !EnvironmentPattern.IsMatch(read.environment))
                throw new ArgumentException("the crash reporting environment must be 1 to 64 letters, digits, '.', '_' or '-'");
            return read;
        }
    }
}
