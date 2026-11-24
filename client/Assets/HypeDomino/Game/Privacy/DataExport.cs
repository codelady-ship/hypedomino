using System;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Privacy
{
    public sealed class DataExport
    {
        public const string Path = "/account/export";

        readonly ApiClient api;

        public DataExport(ApiClient api)
        {
            this.api = api ?? throw new ArgumentNullException(nameof(api));
        }

        public bool Busy { get; private set; }

        public bool Request(Action<DataExportResult> done)
        {
            if (Busy) return false;
            Busy = true;
            api.Post(Path, null, reply => Answer(reply, done));
            return true;
        }

        void Answer(ApiReply reply, Action<DataExportResult> done)
        {
            Busy = false;
            done(DataExportResult.From(reply));
        }
    }
}
