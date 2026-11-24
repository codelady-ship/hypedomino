using System;
using System.Security.Cryptography;
using System.Text;

namespace HypeDomino.Net.Api
{
    public static class RequestMac
    {
        public const string Header = "X-HD-Mac";

        public static string Sign(byte[] key, string method, string pathAndQuery, string access, string body)
        {
            using var hmac = new HMACSHA256(key);
            var digest = hmac.ComputeHash(Encoding.UTF8.GetBytes(method + "\n" + pathAndQuery + "\n" + access + "\n" + body));
            return Convert.ToBase64String(digest).TrimEnd('=').Replace('+', '-').Replace('/', '_');
        }
    }
}
