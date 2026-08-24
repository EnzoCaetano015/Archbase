namespace Example.Application.Services;

public sealed class ExampleService(IExampleRepository repository) : IExampleService
{
    public async Task<IReadOnlyList<ExampleResponse>> ListAsync(
        CancellationToken cancellationToken)
    {
        var entities = await repository.ListAsync(cancellationToken);

        return entities.Select(ExampleResponse.FromEntity).ToArray();
    }
}
