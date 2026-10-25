```dart
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

class AuthProvider extends ChangeNotifier {
  final SharedPreferences _prefs;
  bool _isLoggedIn;

  AuthProvider(SharedPreferences prefs)
    : _prefs = prefs,
      _isLoggedIn =
          (prefs.getString('token') ?? '').isNotEmpty &&
          (prefs.getString('usuario') ?? '').isNotEmpty;

  bool get isLoggedIn => _isLoggedIn;
  String? get token => _isLoggedIn ? _prefs.getString('token') : null;

  Future<void> login(String token, String usuarioJson) async {
    if (token.isEmpty || usuarioJson.isEmpty) {
      throw ArgumentError('Sessão incompleta.');
    }
    final tokenSalvo = await _prefs.setString('token', token);
    final usuarioSalvo = await _prefs.setString('usuario', usuarioJson);
    if (!tokenSalvo || !usuarioSalvo) {
      await logout();
      throw StateError('Não foi possível persistir a sessão.');
    }
    _isLoggedIn = true;
    notifyListeners();
  }

  Future<void> logout() async {
    _isLoggedIn = false;
    notifyListeners();
    final tokenRemovido = await _prefs.remove('token');
    final usuarioRemovido = await _prefs.remove('usuario');
    if (!tokenRemovido || !usuarioRemovido) {
      throw StateError('Não foi possível remover a sessão persistida.');
    }
  }
}
```
