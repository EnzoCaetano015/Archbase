# spring-boot/service@3714

Application service coordinating one feature's persistence and mapping.

- Keep business validation and transaction boundaries in the service layer.
- Inject repositories and mappers through the constructor.
- Avoid servlet request, controller, and response-envelope dependencies.
