using System;
using System.Collections.Generic;
using System.Globalization;
using System.Text;

namespace HypeDomino.UI.Localization
{
    public sealed class MessageFormat
    {
        const string PluralKind = "plural";
        const string SelectKind = "select";
        const char ExactPrefix = '=';
        const string DurationKeyPrefix = "duration.";
        const string DaysName = "days";
        const string HoursName = "hours";
        const string MinutesName = "minutes";
        const string SecondsName = "seconds";

        readonly string source;
        readonly string textLanguage;
        readonly string numberLanguage;
        readonly IReadOnlyDictionary<string, object> args;
        readonly Func<string, string> unitPattern;
        readonly StringBuilder output = new();
        int pos;

        MessageFormat(string source, string textLanguage, string numberLanguage, IReadOnlyDictionary<string, object> args, Func<string, string> unitPattern)
        {
            this.source = source;
            this.textLanguage = textLanguage;
            this.numberLanguage = numberLanguage;
            this.args = args;
            this.unitPattern = unitPattern;
        }

        public static bool TryFormat(string pattern, string textLanguage, string numberLanguage, IReadOnlyDictionary<string, object> args, out string text) =>
            TryFormat(pattern, textLanguage, numberLanguage, args, null, out text);

        public static bool TryFormat(string pattern, string textLanguage, string numberLanguage, IReadOnlyDictionary<string, object> args, Func<string, string> unitPattern, out string text)
        {
            var format = new MessageFormat(pattern ?? string.Empty, textLanguage, numberLanguage, args, unitPattern);
            var ok = format.Message(true, null) && format.pos == format.source.Length;
            text = ok ? format.output.ToString() : null;
            return ok;
        }

        public static string Number(long number, string language)
        {
            var digits = number.ToString(CultureInfo.InvariantCulture);
            var builder = new StringBuilder(digits.Length + digits.Length / 3);
            AppendGrouped(builder, digits, GroupSeparator(language));
            return builder.ToString();
        }

        public static string Decimal(decimal number, string language)
        {
            var digits = number.ToString(CultureInfo.InvariantCulture);
            var point = digits.IndexOf('.');
            var builder = new StringBuilder(digits.Length + digits.Length / 3);
            AppendGrouped(builder, point < 0 ? digits : digits.Substring(0, point), GroupSeparator(language));
            if (point >= 0) builder.Append(DecimalSeparator(language)).Append(digits, point + 1, digits.Length - point - 1);
            return builder.ToString();
        }

        public static string Timestamp(DateTimeOffset instant, TimeZoneInfo zone, string language)
        {
            var local = TimeZoneInfo.ConvertTime(instant, zone);
            var builder = new StringBuilder(18);
            if (language == GameLanguage.English)
            {
                builder.Append(Digits(local.Month)).Append('/').Append(Digits(local.Day)).Append('/').Append(Digits(local.Year)).Append(", ");
            }
            else
            {
                AppendTwoDigits(builder, local.Day).Append('.');
                AppendTwoDigits(builder, local.Month).Append('.').Append(Digits(local.Year)).Append(language == "ru" ? ", " : " ");
            }
            AppendTwoDigits(builder, local.Hour).Append(':');
            AppendTwoDigits(builder, local.Minute);
            return builder.ToString();
        }

        public static string Clock(TimeSpan span)
        {
            var builder = new StringBuilder(8);
            builder.Append(((long)span.TotalHours).ToString(CultureInfo.InvariantCulture)).Append(':');
            AppendTwoDigits(builder, span.Minutes).Append(':');
            AppendTwoDigits(builder, span.Seconds);
            return builder.ToString();
        }

        public static string DurationKey(TimeSpan span)
        {
            if (span.Days > 0) return DurationKeyPrefix + DaysName;
            if (span.Hours > 0) return DurationKeyPrefix + HoursName;
            return DurationKeyPrefix + (span.Minutes > 0 ? MinutesName : SecondsName);
        }

        static void AppendGrouped(StringBuilder builder, string digits, char separator)
        {
            var start = digits[0] == '-' ? 1 : 0;
            if (start == 1) builder.Append('-');
            for (var i = start; i < digits.Length; i++)
            {
                if (i > start && (digits.Length - i) % 3 == 0) builder.Append(separator);
                builder.Append(digits[i]);
            }
        }

        static string Digits(int number) => number.ToString(CultureInfo.InvariantCulture);

        static StringBuilder AppendTwoDigits(StringBuilder builder, int number) =>
            builder.Append((char)('0' + number / 10)).Append((char)('0' + number % 10));

        static char DecimalSeparator(string language) => language is "ru" or "az" or "tr" ? ',' : '.';

        static char GroupSeparator(string language) => language switch
        {
            "ru" => '\u00A0',
            "az" or "tr" => '.',
            _ => ',',
        };

        bool Message(bool emit, string pluralNumber)
        {
            while (pos < source.Length)
            {
                var c = source[pos];
                if (c == '}') return true;
                if (c == '{')
                {
                    if (!Argument(emit)) return false;
                    continue;
                }
                if (emit)
                {
                    if (c == '#' && pluralNumber != null) output.Append(pluralNumber);
                    else output.Append(c);
                }
                pos++;
            }
            return true;
        }

        bool Argument(bool emit)
        {
            pos++;
            var name = Word();
            if (name.Length == 0) return false;
            Spaces();
            if (Take('}')) return !emit || AppendValue(name);
            if (!Take(',')) return false;
            Spaces();
            var kind = Word();
            if (kind != PluralKind && kind != SelectKind) return false;
            Spaces();
            return Take(',') && Branches(name, kind == PluralKind, emit);
        }

        bool AppendValue(string name)
        {
            if (args == null || !args.TryGetValue(name, out var value)) return false;
            switch (value)
            {
                case string text:
                    output.Append(text);
                    return true;
                case decimal number:
                    output.Append(Decimal(number, numberLanguage));
                    return true;
                case DateTimeOffset instant:
                    output.Append(Timestamp(instant, TimeZoneInfo.Local, numberLanguage));
                    return true;
                case TimeSpan span:
                    return AppendDuration(span);
            }
            if (!TryCount(name, out var count)) return false;
            output.Append(Number(count, numberLanguage));
            return true;
        }

        bool AppendDuration(TimeSpan span)
        {
            if (span < TimeSpan.Zero) return false;
            var pattern = unitPattern?.Invoke(DurationKey(span));
            if (pattern == null)
            {
                output.Append(Clock(span));
                return true;
            }
            var parts = new Dictionary<string, object>
            {
                [DaysName] = (long)span.Days,
                [HoursName] = (long)span.Hours,
                [MinutesName] = (long)span.Minutes,
                [SecondsName] = (long)span.Seconds,
            };
            if (!TryFormat(pattern, textLanguage, numberLanguage, parts, out var text)) return false;
            output.Append(text);
            return true;
        }

        bool Branches(string name, bool plural, bool emit)
        {
            var selectors = new List<string>();
            var starts = new List<int>();
            while (true)
            {
                Spaces();
                if (Take('}')) break;
                var selector = Word();
                if (selector.Length == 0) return false;
                Spaces();
                if (!Take('{')) return false;
                selectors.Add(selector);
                starts.Add(pos);
                if (!Message(false, null) || !Take('}')) return false;
            }
            if (selectors.Count == 0) return false;
            if (!emit) return true;
            var end = pos;
            string number = null;
            var chosen = plural ? ChoosePlural(name, selectors, out number) : ChooseSelect(name, selectors);
            if (chosen < 0) return false;
            pos = starts[chosen];
            Message(true, number);
            pos = end;
            return true;
        }

        int ChoosePlural(string name, List<string> selectors, out string number)
        {
            number = null;
            if (!TryCount(name, out var count)) return -1;
            number = Number(count, numberLanguage);
            var exact = selectors.IndexOf(ExactPrefix + count.ToString(CultureInfo.InvariantCulture));
            if (exact >= 0) return exact;
            var category = selectors.IndexOf(PluralRules.Category(textLanguage, count));
            return category >= 0 ? category : selectors.IndexOf(PluralRules.Other);
        }

        int ChooseSelect(string name, List<string> selectors)
        {
            if (args == null || !args.TryGetValue(name, out var value) || value is not string choice) return -1;
            var match = selectors.IndexOf(choice);
            return match >= 0 ? match : selectors.IndexOf(PluralRules.Other);
        }

        bool TryCount(string name, out long count)
        {
            count = 0;
            if (args == null || !args.TryGetValue(name, out var value)) return false;
            switch (value)
            {
                case long number:
                    count = number;
                    return true;
                case int number:
                    count = number;
                    return true;
                default:
                    return false;
            }
        }

        string Word()
        {
            var start = pos;
            while (pos < source.Length)
            {
                var c = source[pos];
                if (c == '{' || c == '}' || c == ',' || char.IsWhiteSpace(c)) break;
                pos++;
            }
            return source.Substring(start, pos - start);
        }

        void Spaces()
        {
            while (pos < source.Length && (source[pos] == ' ' || source[pos] == '\n' || source[pos] == '\t')) pos++;
        }

        bool Take(char c)
        {
            if (pos >= source.Length || source[pos] != c) return false;
            pos++;
            return true;
        }
    }
}
