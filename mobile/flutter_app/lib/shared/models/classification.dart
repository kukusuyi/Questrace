import 'json_helpers.dart';

class Classification {
  const Classification({
    this.subjectId = '',
    this.courseId = '',
    this.status = 'pending',
    this.analysisStale = false,
    this.analysisConfirmed = false,
    this.revision = 0,
    this.subjectName = '',
  });
  final String subjectId, courseId, status, subjectName;
  final bool analysisStale, analysisConfirmed;
  final int revision;
  factory Classification.fromJson(Map<String, dynamic> json) => Classification(
    subjectId: asString(json['subject_id']),
    courseId: asString(json['course_id']),
    status: asString(json['classification_status'], 'pending'),
    analysisStale:
        !json.containsKey('subject_id') || asBool(json['analysis_stale']),
    analysisConfirmed: asBool(json['analysis_confirmed']),
    revision: asInt(json['revision']),
    subjectName: asString(json['subject']),
  );
  Map<String, dynamic> toJson() => {
    'subject_id': subjectId,
    'course_id': courseId,
    'classification_status': status,
    'analysis_stale': analysisStale,
    'analysis_confirmed': analysisConfirmed,
    'revision': revision,
  };
  Classification copyWith({
    String? subjectId,
    String? courseId,
    String? status,
    bool? analysisStale,
    bool? analysisConfirmed,
    int? revision,
    String? subjectName,
  }) => Classification(
    subjectId: subjectId ?? this.subjectId,
    courseId: courseId ?? this.courseId,
    status: status ?? this.status,
    analysisStale: analysisStale ?? this.analysisStale,
    analysisConfirmed: analysisConfirmed ?? this.analysisConfirmed,
    revision: revision ?? this.revision,
    subjectName: subjectName ?? this.subjectName,
  );
  String label(String subject, String chapter) => [
    subjectNames[subjectId] ?? (subject.isEmpty ? '待分类' : subject),
    courseNames[courseId] ?? '',
    chapter,
  ].where((s) => s.isNotEmpty).join(' · ');
  String get statusLabel => status == 'legacy_pending'
      ? '历史分类待复核'
      : status == 'confirmed'
      ? (analysisStale ? '分析待更新' : '')
      : '待分类';
}

const subjectNames = {
  'math_grad': '考研数学',
  'cs408': '408',
  'highschool_math': '高中数学',
  'highschool_geography': '高中地理',
  'highschool_biology': '高中生物',
  'highschool_physics': '高中物理',
  'highschool_chemistry': '高中化学',
};
const courseNames = {
  'data_structures': '数据结构',
  'computer_organization': '计算机组成原理',
  'operating_systems': '操作系统',
  'computer_networks': '计算机网络',
};

class SubjectCourse {
  const SubjectCourse({
    required this.id,
    required this.name,
    required this.chapters,
  });
  final String id, name;
  final List<String> chapters;
  factory SubjectCourse.fromJson(Map<String, dynamic> j) => SubjectCourse(
    id: asString(j['id']),
    name: asString(j['name']),
    chapters: asStringList(j['chapters']),
  );
}

class SubjectItem {
  const SubjectItem({
    this.recommended = true,
    this.educationStage = 'university',
    required this.id,
    required this.name,
    required this.specialized,
    required this.courses,
    required this.chapters,
  });
  final String id, name;
  final bool specialized;
  final bool recommended;
  final String educationStage;
  final List<SubjectCourse> courses;
  final List<String> chapters;
  factory SubjectItem.fromJson(Map<String, dynamic> j) => SubjectItem(
    recommended: j['recommended'] != false,
    educationStage: asString(j['education_stage'], 'university'),
    id: asString(j['id']),
    name: asString(j['name']),
    specialized: asBool(j['specialized']),
    courses: asObjectList(j['courses'], SubjectCourse.fromJson),
    chapters: asStringList(j['chapters']),
  );
}
