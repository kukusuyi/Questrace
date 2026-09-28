import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/network/api_client.dart';
import '../models/classification.dart';

class SubjectSuggestion extends ConsumerStatefulWidget {
  const SubjectSuggestion({
    super.key,
    required this.name,
    required this.onConfirmed,
    required this.onDismissed,
  });
  final String name;
  final void Function(SubjectItem) onConfirmed;
  final VoidCallback onDismissed;
  @override
  ConsumerState<SubjectSuggestion> createState() => _SubjectSuggestionState();
}

class _SubjectSuggestionState extends ConsumerState<SubjectSuggestion> {
  late final TextEditingController controller = TextEditingController(
    text: widget.name,
  );
  bool busy = false;
  String? error;
  @override
  void didUpdateWidget(covariant SubjectSuggestion oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.name != widget.name) {
      controller.text = widget.name;
      error = null;
    }
  }

  @override
  void dispose() {
    controller.dispose();
    super.dispose();
  }

  Future<void> confirm() async {
    if (busy || controller.text.trim().isEmpty) return;
    setState(() {
      busy = true;
      error = null;
    });
    try {
      final r = await ref
          .read(apiClientProvider)
          .post<Map<String, dynamic>>(
            '/api/v1/subjects',
            data: {'name': controller.text.trim()},
          );
      if (!mounted) return;
      widget.onConfirmed(SubjectItem.fromJson(r.data ?? {}));
    } catch (e) {
      if (mounted) setState(() => error = '学科添加失败，请重试');
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      TextField(
        controller: controller,
        enabled: !busy,
        maxLength: 64,
        decoration: InputDecoration(labelText: '学科', errorText: error),
        onChanged: (_) => setState(() {}),
        onSubmitted: (_) => confirm(),
      ),
      const Text('AI 已填写建议学科，确认后加入学科列表。'),
      const SizedBox(height: 8),
      Wrap(
        spacing: 8,
        runSpacing: 8,
        children: [
          FilledButton(
            onPressed: busy || controller.text.trim().isEmpty ? null : confirm,
            child: Text(busy ? '正在确认…' : '确认学科'),
          ),
          TextButton(
            onPressed: busy ? null : widget.onDismissed,
            child: const Text('选择已有学科'),
          ),
        ],
      ),
    ],
  );
}
