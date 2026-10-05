import 'dart:io';
import 'package:flutter_test/flutter_test.dart';
import 'package:nsd/nsd.dart';
import 'package:questrace_flutter/core/discovery/lan_discovery.dart';

Service discovered({
  String? host,
  List<InternetAddress>? addresses,
  int? port = 8080,
}) =>
    Service(
      name: 'Questrace',
      type: '_questrace._tcp',
      host: host,
      port: port,
      addresses: addresses,
    );

void main() {
  test('prefers the resolved address over the advertised .local name', () {
    expect(
      lanHost(discovered(
        host: 'questrace-win-aabbccddeeff.local.',
        addresses: [InternetAddress('192.168.1.104')],
      )),
      '192.168.1.104',
    );
  });

  test('skips loopback and link-local addresses', () {
    expect(
      lanHost(discovered(
        host: 'questrace-mac-aabbccddeeff.local.',
        addresses: [
          InternetAddress('127.0.0.1'),
          InternetAddress('169.254.3.4'),
          InternetAddress('10.0.0.5'),
        ],
      )),
      '10.0.0.5',
    );
  });

  test('uses a non-private address only when nothing better exists', () {
    expect(
      lanHost(discovered(
        host: 'questrace-win-aabbccddeeff.local.',
        addresses: [InternetAddress('203.0.113.7')],
      )),
      '203.0.113.7',
    );
  });

  test('falls back to the hostname without its trailing dot', () {
    expect(
      lanHost(discovered(host: 'questrace-win-aabbccddeeff.local.')),
      'questrace-win-aabbccddeeff.local',
    );
    expect(lanHost(discovered()), isNull);
    expect(lanHost(discovered(host: '', addresses: const [])), isNull);
  });

  test('recognizes the private IPv4 ranges Questrace advertises', () {
    for (final address in ['10.0.0.1', '172.16.0.1', '172.31.255.254', '192.168.0.1']) {
      expect(isPrivateIPv4(address), isTrue, reason: address);
    }
    for (final address in ['172.15.0.1', '172.32.0.1', '192.169.0.1', '203.0.113.7', '::1', 'not-an-ip']) {
      expect(isPrivateIPv4(address), isFalse, reason: address);
    }
  });
}
