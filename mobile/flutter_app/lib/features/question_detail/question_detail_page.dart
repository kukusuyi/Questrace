import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../shared/widgets/async_value_view.dart';
import '../../shared/widgets/latex_block.dart';
import '../../shared/widgets/remote_image_card.dart';
import '../question_list/question_repository.dart';

class QuestionDetailPage extends ConsumerWidget {
  const QuestionDetailPage({super.key, required this.questionId});

  final int questionId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final detail = ref.watch(questionDetailProvider(questionId));

    return Scaffold(
      appBar: AppBar(
        title: const Text('错题详情'),
        actions: [
          IconButton(
            onPressed: () => context.push('/questions/$questionId/edit'),
            icon: const Icon(Icons.edit_outlined),
            tooltip: '编辑错题',
          ),
          IconButton(
            onPressed: () => context.push('/questions/$questionId/similar'),
            icon: const Icon(Icons.hub_outlined),
            tooltip: '查看相似题',
          ),
        ],
      ),
      body: AsyncValueView(
        value: detail,
        builder: (data) {
          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              _SectionCard(
                title: '题目主干',
                child: LatexBlock(
                  data.questionCore,
                  allowHorizontalScroll: true,
                ),
              ),
              const SizedBox(height: 16),
              if (data.sourceImageUrl.isNotEmpty)
                RemoteImageCard(imageUrl: data.sourceImageUrl),
              if (data.sourceImageUrl.isNotEmpty) const SizedBox(height: 16),
              _SectionCard(
                title: '基本信息',
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '${data.classification.label(data.subject, data.chapter)} ${data.classification.statusLabel}',
                    ),
                    const SizedBox(height: 8),
                    Text('掌握状态：${data.masteryStatus.label}'),
                    const SizedBox(height: 8),
                    Text('难度：${data.difficultyLevel} / 5'),
                  ],
                ),
              ),
              const SizedBox(height: 16),
              const SizedBox(height: 16),
              _SectionCard(
                title: '标准解',
                child: LatexBlock(
                  data.standardSolution,
                  allowHorizontalScroll: true,
                ),
              ),
              const SizedBox(height: 16),
              _SectionCard(
                title: '错误解',
                child: LatexBlock(
                  data.wrongSolution,
                  allowHorizontalScroll: true,
                ),
              ),
              const SizedBox(height: 16),
              _SectionCard(
                title: 'AI 总结',
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('语义总结：${data.semanticSummary}'),
                    const SizedBox(height: 12),
                    Text('错误总结：${data.mistakeSummary}'),
                  ],
                ),
              ),
              const SizedBox(height: 16),
            ],
          );
        },
      ),
    );
  }
}

class _SectionCard extends StatelessWidget {
  const _SectionCard({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              style: Theme.of(
                context,
              ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 12),
            child,
          ],
        ),
      ),
    );
  }
}
