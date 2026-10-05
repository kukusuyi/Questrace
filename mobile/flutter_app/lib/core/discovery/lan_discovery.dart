import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:nsd/nsd.dart';

class LanComputer {
  const LanComputer(this.id, this.name, this.url);
  final String id, name, url;
}

/// Address the phone should dial for one discovered service.
///
/// Questrace answers each mDNS query with A records for the interface that
/// received it, and every platform plugin exposes that already resolved address
/// in [Service.addresses] (Android: `NsdServiceInfo.host.hostAddress`; Apple:
/// the `NetService` addresses). Dialling the address directly avoids asking the
/// phone to resolve the advertised `.local` name a second time; on Android and
/// iOS that second resolution is the step that fails most often. The hostname
/// remains the fallback for platforms that only report a name.
///
/// Returns null when neither an address nor a hostname is usable, so the caller
/// can skip the service.
String? lanHost(Service service) {
  final addresses = service.addresses ?? const <InternetAddress>[];
  String? otherIPv4;
  for (final address in addresses) {
    if (address.type != InternetAddressType.IPv4 || address.isLoopback) {
      continue;
    }
    final ip = address.address;
    // A link-local address only works on the phone's own link.
    if (ip.startsWith('169.254.')) {
      continue;
    }
    if (isPrivateIPv4(ip)) {
      return ip;
    }
    otherIPv4 ??= ip;
  }
  if (otherIPv4 != null) {
    return otherIPv4;
  }
  final host = service.host;
  if (host == null || host.isEmpty) {
    return null;
  }
  return host.replaceFirst(RegExp(r'\.$'), '');
}

/// True for the RFC 1918 ranges Questrace advertises.
bool isPrivateIPv4(String ip) {
  final parts = ip.split('.');
  if (parts.length != 4) {
    return false;
  }
  final first = int.tryParse(parts[0]);
  final second = int.tryParse(parts[1]);
  if (first == null || second == null) {
    return false;
  }
  if (first == 10) {
    return true;
  }
  if (first == 192 && second == 168) {
    return true;
  }
  return first == 172 && second >= 16 && second <= 31;
}

class LanDiscovery {
  Future<List<LanComputer>> scan() async {
    final discovery = await startDiscovery('_questrace._tcp');
    try {
      await Future<void>.delayed(const Duration(seconds: 4));
      final found = <String, LanComputer>{};
      for (final service in discovery.services) {
        final id = utf8.decode(service.txt?['id'] ?? [], allowMalformed: true);
        final host = lanHost(service);
        if (!RegExp(r'^[a-f0-9]{12}$').hasMatch(id) ||
            host == null ||
            service.port == null) {
          continue;
        }
        final url = Uri(
          scheme: 'http',
          host: host,
          port: service.port!,
        ).toString();
        found[id] = LanComputer(id, service.name ?? host, url);
      }
      return found.values.toList();
    } finally {
      await stopDiscovery(discovery);
    }
  }
}
