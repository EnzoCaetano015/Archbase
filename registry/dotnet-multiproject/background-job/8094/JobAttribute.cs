namespace Example.Jobs.Attributes;

[AttributeUsage(AttributeTargets.Class, Inherited = false)]
public sealed class JobAttribute(string cron, string? jobId = null) : Attribute
{
    public string Cron { get; } = cron;
    public string? JobId { get; } = jobId;
}
