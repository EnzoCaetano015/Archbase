using Example.Application;
using Example.Application.Interfaces;
using Example.Infrastructure;
using Example.Infrastructure.Interfaces;

namespace Example.Api.Config;

internal static class DependencyInjection
{
    internal static IServiceCollection AddApplicationDependencies(
        this IServiceCollection services,
        IConfiguration configuration)
    {
        services.AddScoped<IItemService, ItemService>();
        services.AddScoped<IItemRepository, ItemRepository>();
        return services;
    }
}
