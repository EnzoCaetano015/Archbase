using System.ComponentModel.DataAnnotations;

namespace Example.Contracts.Requests;

public sealed class CreateItemRequest
{
    [Required]
    [MaxLength(100)]
    public required string Name { get; init; }

    public DateTime CreatedAt { get; init; } = DateTime.UtcNow;
}
