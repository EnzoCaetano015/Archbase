import { ValidationPipe } from "@nestjs/common"
import { NestFactory } from "@nestjs/core"
import { DocumentBuilder, SwaggerModule } from "@nestjs/swagger"
import { AppModule } from "./app.module"
import { ApiExceptionFilter } from "./common/http/api-exception.filter"
import { ResponseEnvelopeInterceptor } from "./common/http/response-envelope.interceptor"

async function bootstrap() {
  const app = await NestFactory.create(AppModule)

  app.useGlobalPipes(new ValidationPipe({ transform: true, whitelist: true }))
  app.useGlobalFilters(new ApiExceptionFilter())
  app.useGlobalInterceptors(new ResponseEnvelopeInterceptor())

  const swagger = new DocumentBuilder().setTitle("Example API").setVersion("1.0").build()
  SwaggerModule.setup("docs", app, SwaggerModule.createDocument(app, swagger))

  await app.listen(process.env.PORT ?? 3000)
}

void bootstrap()
