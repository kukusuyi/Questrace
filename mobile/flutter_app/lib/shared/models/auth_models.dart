import 'json_helpers.dart';

class LoginPayload {
  const LoginPayload({required this.username, required this.password});

  final String username;
  final String password;

  Map<String, dynamic> toJson() {
    return {'username': username, 'password': password};
  }
}

class RegisterPayload {
  const RegisterPayload({
    required this.educationStage,
    required this.username,
    required this.password,
    required this.email,
  });

  final String username;
  final String password;
  final String email;
  final String educationStage;

  Map<String, dynamic> toJson() {
    return {
      'username': username,
      'password': password,
      'email': email,
      'education_stage': educationStage,
    };
  }
}

class AuthSession {
  const AuthSession({
    required this.userId,
    required this.username,
    required this.token,
  });

  final int userId;
  final String username;
  final String token;

  factory AuthSession.fromJson(Map<String, dynamic> json) {
    return AuthSession(
      userId: asInt(json['user_id']),
      username: asString(json['username']),
      token: asString(json['token']),
    );
  }

  Map<String, dynamic> toJson() {
    return {'user_id': userId, 'username': username, 'token': token};
  }
}
