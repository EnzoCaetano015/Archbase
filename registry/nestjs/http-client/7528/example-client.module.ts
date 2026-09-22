import { HttpModule } from "@nestjs/axios"
import { Module } from "@nestjs/common"
import { ConfigService } from "@nestjs/config"
import { CatalogClient } from "./catalog-client.service"

@Module({
  imports: [
    HttpModule.registerAsync({
      inject: [ConfigService],
      useFactory: (config: ConfigService) => ({
        baseURL: config.getOrThrow<string>("CATALOG_API_URL"),
        headers: { Authorization: `Bearer ${config.getOrThrow<string>("CATALOG_API_TOKEN")}` },
      }),
    }),
  ],
  providers: [CatalogClient],
  exports: [CatalogClient],
})
export class CatalogClientModule {}
