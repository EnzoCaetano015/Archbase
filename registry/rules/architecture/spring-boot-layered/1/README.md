# Layered Spring Boot API architecture

This variant is derived from the Simple Java API reference: Spring Boot 4 on Java 21, Spring MVC controllers, transactional services, Spring Data JPA repositories, audited entities, validated record DTOs, MapStruct mappers, stateless JWT security, and global response advice.

The architecture keeps the conventional package layout while making dependency direction explicit: web delegates to application services, services coordinate persistence and mapping, repositories isolate JPA, and domain contracts remain independent of web behavior.

When creating or modifying matching code, resolve the nearest local scope first and inspect the associated pattern for its required structure, allowed changes, and preserved responsibilities.
