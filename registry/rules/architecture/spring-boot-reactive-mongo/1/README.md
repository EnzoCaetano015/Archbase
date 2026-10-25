# Spring Boot reactive MongoDB architecture

Derived from the supplied Horizon Java API: Java 22, Spring Boot 3.3.3, Spring MVC, reactive Spring Data MongoDB, Reactor, Lombok, and Springdoc OpenAPI. Versions describe the reference project rather than a dependency upgrade recommendation.

Keep controller, domain, exception, repository, request, response, service, and service/impl packages. Controllers depend on service interfaces; implementations compose repository publishers and map documents into response projections. Resolve the nearest project scope and inspect its pattern before editing code.

The eight patterns are independent structural examples, not a feature installer or a runnable application. The service pattern bundles its interface and implementation. Request and response contracts remain separate from MongoDB documents. Unlike the existing JPA variant, this architecture does not introduce MapStruct, JWT security, auditing, or global response envelopes.

Examples generalize Horizon's resource names and package. The repository example makes the partial-update operator explicit with `$set`; the service interface omits an unused domain import. Lombok null checks from the reference are retained as a structural convention and do not provide complete HTTP input validation. The custom update query assumes MongoDB ObjectId-backed identifiers; adapt it when the document uses another identity representation.

The reference uses the MVC web starter and returns Reactor publishers. This does not imply a fully WebFlux application. No Maven project, database settings, secrets, or business-specific data are installed. Java compilation and MongoDB integration require a consuming project and are outside catalog validation.
