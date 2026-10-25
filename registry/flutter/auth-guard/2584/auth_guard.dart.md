```dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'notificacao_provider.dart';

class AuthGuard extends StatefulWidget {
  final WidgetBuilder builder;
  const AuthGuard({super.key, required this.builder});

  @override
  State<AuthGuard> createState() => _AuthGuardState();
}

class _AuthGuardState extends State<AuthGuard> {
  bool _redirectAgendado = false;

  @override
  Widget build(BuildContext context) {
    if (context.watch<AuthProvider>().isLoggedIn) {
      _redirectAgendado = false;
      return widget.builder(context);
    }
    if (!_redirectAgendado) {
      _redirectAgendado = true;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        if (!context.read<AuthProvider>().isLoggedIn) {
          Navigator.of(context).pushReplacementNamed('/autenticacao');
        }
      });
    }
    return const SizedBox.shrink();
  }
}
```
