```dart
enum RoleEnum {
  administrador(1),
  membro(2);

  final int value;
  const RoleEnum(this.value);

  static String nome(int id) {
    switch (id) {
      case 1:
        return 'Administrador';
      case 2:
        return 'Membro';
      default:
        return 'Desconhecido';
    }
  }
}
```
