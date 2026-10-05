import '../shared/models/classification.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/auth/auth_controller.dart';
import '../features/auth/auth_page.dart';
import '../features/dashboard/dashboard_page.dart';
import '../features/question_create/question_create_page.dart';
import '../features/question_detail/question_detail_page.dart';
import '../features/question_edit/question_edit_page.dart';
import '../features/question_list/question_list_page.dart';
import '../features/question_review/ai_review_page.dart';
import '../features/question_review/ocr_review_page.dart';
import '../features/question_upload/question_upload_page.dart';
import '../features/settings/settings_page.dart';
import '../features/similar_question/similar_question_page.dart';
import '../features/tag_manage/tag_manage_page.dart';
import '../shared/models/common_models.dart';
import '../shared/models/question_models.dart';
import '../features/review/review_page.dart';
import 'app_shell.dart';
import 'tab_navigation_intent.dart';

final appRouterProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authControllerProvider);

  return GoRouter(
    initialLocation: '/dashboard',
    redirect: (context, state) {
      final isAuthRoute = state.matchedLocation == '/auth';
      if (!authState.isAuthenticated && !isAuthRoute) {
        return '/auth';
      }
      if (authState.isAuthenticated && isAuthRoute) {
        return '/dashboard';
      }
      return null;
    },
    routes: [
      GoRoute(path: '/auth', builder: (context, state) => const AuthPage()),
      ShellRoute(
        builder: (context, state, child) {
          return AppShell(
            currentLocation: state.uri.path,
            tabNavigationIntent: state.extra is TabNavigationIntent
                ? state.extra as TabNavigationIntent
                : null,
            child: child,
          );
        },
        routes: [
          GoRoute(
            path: '/reviews',
            builder: (context, state) => const ReviewPage(),
          ),
          GoRoute(
            path: '/reviews/:id',
            builder: (context, state) =>
                ReviewPage(sessionId: int.parse(state.pathParameters['id']!)),
          ),
          GoRoute(
            path: '/dashboard',
            pageBuilder: (context, state) =>
                _buildTabPage(state: state, child: const DashboardPage()),
          ),
          GoRoute(
            path: '/questions',
            pageBuilder: (context, state) => _buildTabPage(
              state: state,
              child: QuestionListPage(
                initialFilter: _questionFilterFromState(state),
                activeTagName: state.uri.queryParameters['tag_name'] ?? '',
              ),
            ),
          ),
          GoRoute(
            path: '/questions/create',
            pageBuilder: (context, state) =>
                _buildTabPage(state: state, child: const QuestionCreatePage()),
          ),
          GoRoute(
            path: '/questions/upload',
            pageBuilder: (context, state) =>
                _buildTabPage(state: state, child: const QuestionUploadPage()),
          ),
          GoRoute(
            path: '/questions/ocr-review',
            pageBuilder: (context, state) =>
                _buildTabPage(state: state, child: const OcrReviewPage()),
          ),
          GoRoute(
            path: '/questions/ai-review',
            pageBuilder: (context, state) =>
                _buildTabPage(state: state, child: const AiReviewPage()),
          ),
          GoRoute(
            path: '/questions/:id',
            builder: (context, state) => QuestionDetailPage(
              questionId: int.parse(state.pathParameters['id']!),
            ),
          ),
          GoRoute(
            path: '/questions/:id/edit',
            builder: (context, state) => QuestionEditPage(
              questionId: int.parse(state.pathParameters['id']!),
            ),
          ),
          GoRoute(
            path: '/questions/:id/similar',
            builder: (context, state) => SimilarQuestionPage(
              questionId: int.parse(state.pathParameters['id']!),
            ),
          ),
          GoRoute(
            path: '/tags',
            pageBuilder: (context, state) =>
                _buildTabPage(state: state, child: const TagManagePage()),
          ),
          GoRoute(
            path: '/settings',
            pageBuilder: (context, state) =>
                _buildTabPage(state: state, child: const SettingsPage()),
          ),
        ],
      ),
    ],
  );
});

Page<void> _buildTabPage({
  required GoRouterState state,
  required Widget child,
}) {
  return NoTransitionPage<void>(key: state.pageKey, child: child);
}

ListQuestionFilter _questionFilterFromState(GoRouterState state) {
  final query = state.uri.queryParameters;

  return ListQuestionFilter(
    keyword: query['keyword'],
    classification: Classification(
      subjectId: query['subject_id'] ?? '',
      courseId: query['course_id'] ?? '',
      status: query['classification_status'] ?? '',
    ),
    subject: query['subject'],
    chapter: query['chapter'],
    page: int.tryParse(query['page'] ?? '1') ?? 1,
    masteryStatus: _masteryStatusFromValue(query['mastery_status']),
    sourceType: _sourceTypeFromValue(query['source_type']),
    tagIds: _parseTagIds(query['tag_ids']),
  );
}

MasteryStatus? _masteryStatusFromValue(String? value) {
  if (value == null || value.isEmpty) {
    return null;
  }

  for (final item in MasteryStatus.values) {
    if (item.value == value) {
      return item;
    }
  }

  return null;
}

SourceType? _sourceTypeFromValue(String? value) {
  if (value == null || value.isEmpty) {
    return null;
  }

  for (final item in SourceType.values) {
    if (item.value == value) {
      return item;
    }
  }

  return null;
}

List<int> _parseTagIds(String? raw) {
  if (raw == null || raw.trim().isEmpty) {
    return const <int>[];
  }

  return raw
      .split(',')
      .map((item) => int.tryParse(item.trim()))
      .whereType<int>()
      .toList();
}
