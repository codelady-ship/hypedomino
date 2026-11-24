using System;
using System.Collections.Generic;
using UnityEngine;

namespace HypeDomino.Game.Friends
{
    public enum FriendPresence
    {
        Offline,
        Online,
        InMatch,
    }

    public readonly struct Friend
    {
        public Friend(string account, string nickname, FriendPresence presence)
        {
            Account = account;
            Nickname = nickname;
            Presence = presence;
        }

        public string Account { get; }
        public string Nickname { get; }
        public FriendPresence Presence { get; }
    }

    [Serializable]
    public sealed class FriendReply
    {
        public string account_id;
        public string nickname;
        public string presence;
    }

    [Serializable]
    public sealed class FriendsReply
    {
        public FriendReply[] friends;
    }

    public static class FriendList
    {
        public const string Path = "/friends";
        public const string Online = "online";
        public const string InMatch = "in_match";

        public static bool TryRead(string json, List<Friend> friends)
        {
            if (!TryParse(json, out var reply) || !AllNamed(reply.friends)) return false;
            friends.Clear();
            if (reply.friends == null) return true;
            foreach (var friend in reply.friends)
            {
                friends.Add(new Friend(friend.account_id, friend.nickname ?? string.Empty, PresenceOf(friend.presence)));
            }
            return true;
        }

        public static FriendPresence PresenceOf(string presence) => presence switch
        {
            Online => FriendPresence.Online,
            InMatch => FriendPresence.InMatch,
            _ => FriendPresence.Offline,
        };

        static bool TryParse(string json, out FriendsReply reply)
        {
            reply = null;
            if (string.IsNullOrEmpty(json)) return false;
            try
            {
                reply = JsonUtility.FromJson<FriendsReply>(json);
            }
            catch (ArgumentException)
            {
                return false;
            }
            return reply != null;
        }

        static bool AllNamed(FriendReply[] friends)
        {
            if (friends == null) return true;
            foreach (var friend in friends)
            {
                if (friend == null || string.IsNullOrEmpty(friend.account_id)) return false;
            }
            return true;
        }
    }
}
