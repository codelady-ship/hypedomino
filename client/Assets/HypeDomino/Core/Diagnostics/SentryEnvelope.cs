using System;
using System.Collections.Generic;
using System.Globalization;
using System.Text;
using UnityEngine;

namespace HypeDomino.Core.Diagnostics
{
    public sealed class SentryEnvelope
    {
        public const string Platform = "csharp";
        public const string Logger = "unity";
        public const string SdkName = "hypedomino.unity";
        public const string InAppPrefix = "HypeDomino.";
        const string TimestampFormat = "yyyy-MM-dd'T'HH:mm:ss.fff'Z'";
        const string LocationOpen = "(at ";

        readonly string release;
        readonly string environment;
        readonly string operatingSystem;
        readonly string deviceModel;

        public SentryEnvelope(string release, string environment, string operatingSystem, string deviceModel)
        {
            if (string.IsNullOrEmpty(release)) throw new ArgumentException("release required", nameof(release));
            this.release = release;
            this.environment = environment;
            this.operatingSystem = operatingSystem;
            this.deviceModel = deviceModel;
        }

        public string Build(ErrorReport report, string eventId, DateTime utcNow)
        {
            var timestamp = utcNow.ToString(TimestampFormat, CultureInfo.InvariantCulture);
            var json = new StringBuilder(1024);
            json.Append("{\"event_id\":").AppendString(eventId).Append(",\"sent_at\":").AppendString(timestamp).Append("}\n");
            json.Append("{\"type\":\"event\"}\n");
            json.Append("{\"event_id\":").AppendString(eventId);
            json.Append(",\"timestamp\":").AppendString(timestamp);
            json.Append(",\"platform\":").AppendString(Platform);
            json.Append(",\"logger\":").AppendString(Logger);
            json.Append(",\"level\":\"error\"");
            json.Append(",\"release\":").AppendString(release);
            if (!string.IsNullOrEmpty(environment)) json.Append(",\"environment\":").AppendString(environment);
            json.Append(",\"sdk\":{\"name\":").AppendString(SdkName).Append(",\"version\":").AppendString(release).Append(",\"settings\":{\"infer_ip\":\"never\"}}");
            json.Append(",\"tags\":{\"log_type\":").AppendString(report.Type.ToString()).Append('}');
            AppendContexts(json);
            AppendException(json, report);
            json.Append("}\n");
            return json.ToString();
        }

        void AppendContexts(StringBuilder json)
        {
            json.Append(",\"contexts\":{\"os\":{\"raw_description\":").AppendString(operatingSystem ?? string.Empty);
            json.Append("},\"device\":{\"model\":").AppendString(deviceModel ?? string.Empty).Append("}}");
        }

        static void AppendException(StringBuilder json, ErrorReport report)
        {
            var message = ErrorScrub.Scrub(report.Message);
            var type = report.Type.ToString();
            var value = message;
            if (report.Type == LogType.Exception && TrySplitException(message, out var exceptionType, out var exceptionValue))
            {
                type = exceptionType;
                value = exceptionValue;
            }
            json.Append(",\"exception\":{\"values\":[{\"type\":").AppendString(type).Append(",\"value\":").AppendString(value);
            json.Append(",\"stacktrace\":{\"frames\":[");
            var frames = Frames(ErrorScrub.Scrub(report.StackTrace));
            for (var i = frames.Count - 1; i >= 0; i--)
            {
                if (i < frames.Count - 1) json.Append(',');
                AppendFrame(json, frames[i]);
            }
            json.Append("]}}]}");
        }

        static void AppendFrame(StringBuilder json, string line)
        {
            var open = line.IndexOf(" (", StringComparison.Ordinal);
            var function = open < 0 ? line : line.Substring(0, open);
            json.Append("{\"function\":").AppendString(function);
            json.Append(",\"in_app\":").Append(function.StartsWith(InAppPrefix, StringComparison.Ordinal) ? "true" : "false");
            if (TryLocation(line, out var file, out var lineNumber))
                json.Append(",\"filename\":").AppendString(file).Append(",\"lineno\":").Append(lineNumber.ToString(CultureInfo.InvariantCulture));
            json.Append('}');
        }

        static bool TryLocation(string line, out string file, out int lineNumber)
        {
            file = null;
            lineNumber = 0;
            var start = line.LastIndexOf(LocationOpen, StringComparison.Ordinal);
            if (start < 0 || !line.EndsWith(")", StringComparison.Ordinal)) return false;
            var location = line.Substring(start + LocationOpen.Length, line.Length - start - LocationOpen.Length - 1);
            var colon = location.LastIndexOf(':');
            if (colon <= 0 || location[0] == '<') return false;
            if (!int.TryParse(location.Substring(colon + 1), NumberStyles.None, CultureInfo.InvariantCulture, out lineNumber) || lineNumber < 1) return false;
            file = location.Substring(0, colon);
            return true;
        }

        static bool TrySplitException(string message, out string type, out string value)
        {
            type = null;
            value = message;
            var separator = message.IndexOf(": ", StringComparison.Ordinal);
            if (separator <= 0) return false;
            var head = message.Substring(0, separator);
            if (head.IndexOf(' ') >= 0 || head.IndexOf('\n') >= 0) return false;
            type = head;
            value = message.Substring(separator + 2);
            return true;
        }

        static List<string> Frames(string stackTrace)
        {
            var frames = new List<string>();
            if (string.IsNullOrEmpty(stackTrace)) return frames;
            foreach (var raw in stackTrace.Split('\n'))
            {
                var line = raw.Trim();
                if (line.Length > 0) frames.Add(line);
            }
            return frames;
        }
    }

    static class JsonText
    {
        public static StringBuilder AppendString(this StringBuilder json, string text)
        {
            json.Append('"');
            foreach (var character in text)
            {
                switch (character)
                {
                    case '"': json.Append("\\\""); break;
                    case '\\': json.Append("\\\\"); break;
                    case '\n': json.Append("\\n"); break;
                    case '\r': json.Append("\\r"); break;
                    case '\t': json.Append("\\t"); break;
                    default:
                        if (character < ' ') json.Append("\\u").Append(((int)character).ToString("x4", CultureInfo.InvariantCulture));
                        else json.Append(character);
                        break;
                }
            }
            return json.Append('"');
        }
    }
}
