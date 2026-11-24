namespace HypeDomino.Core.DeviceRisk
{
    public readonly struct DeviceFacts
    {
        public readonly string Fingerprint;
        public readonly string Model;
        public readonly string Manufacturer;
        public readonly string Hardware;
        public readonly string Product;
        public readonly string Tags;

        public DeviceFacts(string fingerprint, string model, string manufacturer, string hardware, string product, string tags)
        {
            Fingerprint = fingerprint ?? "";
            Model = model ?? "";
            Manufacturer = manufacturer ?? "";
            Hardware = hardware ?? "";
            Product = product ?? "";
            Tags = tags ?? "";
        }
    }
}
