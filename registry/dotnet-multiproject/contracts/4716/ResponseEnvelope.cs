namespace Example.Contracts.Responses;

public sealed record ResponseEnvelope<T>(bool Success, T? Data = default, string? Error = null);
