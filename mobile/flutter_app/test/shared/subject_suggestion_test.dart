import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:questrace_flutter/core/network/api_client.dart';
import 'package:questrace_flutter/shared/models/classification.dart';
import 'package:questrace_flutter/shared/widgets/subject_suggestion.dart';

void main() {
  testWidgets(
    'AI subject is prefilled, editable, and saved only on confirmation',
    (tester) async {
      var requests = 0;
      SubjectItem? selected;
      final dio = Dio()
        ..interceptors.add(
          InterceptorsWrapper(
            onRequest: (options, handler) {
              requests++;
              expect(options.data['name'], '机械原理');
              if (requests == 1) {
                handler.reject(DioException(requestOptions: options));
                return;
              }
              handler.resolve(
                Response(
                  requestOptions: options,
                  statusCode: 200,
                  data: {
                    'id': 'custom_mechanical',
                    'name': '机械原理',
                    'chapters': [],
                    'courses': [],
                  },
                ),
              );
            },
          ),
        );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [apiClientProvider.overrideWithValue(dio)],
          child: MaterialApp(
            home: Scaffold(
              body: SubjectSuggestion(
                name: '材料力学',
                onConfirmed: (s) => selected = s,
                onDismissed: () {},
              ),
            ),
          ),
        ),
      );
      expect(find.text('材料力学'), findsOneWidget);
      expect(requests, 0);
      await tester.enterText(find.byType(TextField), '机械原理');
      await tester.tap(find.text('确认学科'));
      await tester.pumpAndSettle();
      expect(selected, isNull);
      expect(find.text('机械原理'), findsOneWidget);
      expect(find.text('学科添加失败，请重试'), findsOneWidget);
      await tester.tap(find.text('确认学科'));
      await tester.pumpAndSettle();
      expect(selected?.id, 'custom_mechanical');
      expect(requests, 2);
    },
  );
}
