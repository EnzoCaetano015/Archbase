namespace Example.Contracts.Models;

public sealed class Item
{
    public required int Id { get; init; }
    public required string Name { get; init; }
    public DateTime CreatedAt { get; init; }
}
