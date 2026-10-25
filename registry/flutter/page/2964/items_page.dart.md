```dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../schemas/item.model.dart';
import '../../services/items_service.dart';
import '../../shared/cards/item_card.dart';

class ItemsPage extends StatefulWidget {
  const ItemsPage({super.key});

  @override
  State<ItemsPage> createState() => _ItemsPageState();
}

class _ItemsPageState extends State<ItemsPage> {
  List<Item> _originais = [];
  String _filtro = '';
  bool _carregando = true;
  String? _erro;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _carregar();
    });
  }

  Future<void> _carregar() async {
    setState(() {
      _carregando = true;
      _erro = null;
    });
    try {
      final items = await context.read<ItemsService>().listar();
      if (!mounted) return;
      setState(() {
        _originais = items;
        _carregando = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _erro = 'Não foi possível carregar os itens.';
        _carregando = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final filtrados = _originais
        .where(
          (item) => item.titulo.toLowerCase().contains(_filtro.toLowerCase()),
        )
        .toList();
    return Scaffold(
      appBar: AppBar(title: const Text('Itens')),
      body: Column(
        children: [
          TextField(
            decoration: const InputDecoration(labelText: 'Filtrar'),
            onChanged: (value) => setState(() => _filtro = value),
          ),
          Expanded(child: _conteudo(filtrados)),
        ],
      ),
    );
  }

  Widget _conteudo(List<Item> items) {
    if (_carregando) return const Center(child: CircularProgressIndicator());
    if (_erro != null) {
      return Center(
        child: TextButton(onPressed: _carregar, child: Text(_erro!)),
      );
    }
    if (items.isEmpty)
      return const Center(child: Text('Nenhum item encontrado.'));
    return LayoutBuilder(
      builder: (context, constraints) {
        final colunas = (constraints.maxWidth / 380).floor();
        Widget card(int index) => ItemCard(
          titulo: items[index].titulo,
          onTap: () =>
              Navigator.of(context).pushNamed('/item/${items[index].id}'),
        );
        if (colunas > 1) {
          return GridView.builder(
            gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: colunas,
              childAspectRatio: 3,
            ),
            itemCount: items.length,
            itemBuilder: (_, index) => card(index),
          );
        }
        return ListView.builder(
          itemCount: items.length,
          itemBuilder: (_, index) => card(index),
        );
      },
    );
  }
}
```
