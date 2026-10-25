```csharp
using System;
using System.Text;
using Example.Repository;
using Example.Repository.Interfaces;
using Example.Service;
using Example.Service.Interfaces;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.AspNetCore.Builder;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.IdentityModel.Tokens;

var builder = WebApplication.CreateBuilder(args);
string Required(string key) => builder.Configuration[key]
    ?? throw new InvalidOperationException($"Configuração ausente: {key}");

var origins = new[] { Required("Rota:Front"), Required("Rota:Admin") };
builder.Services.AddCors(options => options.AddPolicy("FrontendOrigins", policy =>
    policy.WithOrigins(origins).AllowAnyMethod().AllowAnyHeader()));

builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options => options.TokenValidationParameters = new TokenValidationParameters
    {
        ValidateIssuer = true,
        ValidateAudience = true,
        ValidateLifetime = true,
        ValidateIssuerSigningKey = true,
        ValidIssuer = Required("Jwt:Issuer"),
        ValidAudience = Required("Jwt:Audience"),
        IssuerSigningKey = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(Required("Jwt:Key")))
    });
builder.Services.AddAuthorization();
builder.Services.AddDbContext<ApplicationDbContext>(options =>
    options.UseSqlServer(Required("ConnectionStrings:DefaultConnection")));
builder.Services.AddScoped<IItemRepository, ItemRepository>();
builder.Services.AddScoped<IItemService, ItemService>();

var externalUrl = new Uri(Required("ServicoExterno:Url"), UriKind.Absolute);
builder.Services.AddHttpClient<IAnaliseExternoService, AnaliseExternoService>(client =>
{
    client.BaseAddress = externalUrl;
    client.Timeout = TimeSpan.FromSeconds(30);
});
builder.Services.AddControllers();

var app = builder.Build();
app.UseHttpsRedirection();
app.UseCors("FrontendOrigins");
app.UseAuthentication();
app.UseAuthorization();
app.MapControllers();
app.Run();
```
