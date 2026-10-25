```dart
import 'package:flutter/material.dart';

class PreferenciaTema {
  static final ValueNotifier<Brightness> tema = ValueNotifier(Brightness.light);

  static void setTema() {
    tema.value = WidgetsBinding.instance.platformDispatcher.platformBrightness;
  }

  static ThemeData build(Brightness brightness) =>
      ThemeData(brightness: brightness, colorSchemeSeed: Colors.indigo);
}
```
