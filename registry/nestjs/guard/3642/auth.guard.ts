import { CanActivate, ExecutionContext, Injectable, UnauthorizedException } from "@nestjs/common"
import { Reflector } from "@nestjs/core"
import { AuthService } from "./auth.service"
import { IS_PUBLIC_KEY } from "./public-route.decorator"

@Injectable()
export class AuthGuard implements CanActivate {
  constructor(
    private readonly reflector: Reflector,
    private readonly authService: AuthService,
  ) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const isPublic = this.reflector.getAllAndOverride<boolean>(IS_PUBLIC_KEY, [
      context.getHandler(),
      context.getClass(),
    ])
    if (isPublic) return true

    const request = context.switchToHttp().getRequest<{ headers: { authorization?: string }; user?: unknown }>()
    const token = request.headers.authorization?.replace(/^Bearer\s+/i, "")
    if (!token) throw new UnauthorizedException("Missing access token")

    request.user = await this.authService.verifyToken(token)
    return true
  }
}
