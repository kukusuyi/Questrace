"""Run against a native prebuilt server; all data lives in a temporary directory."""
from contextlib import ExitStack
import json
import pathlib
import queue
import socket
import subprocess
import sys
import tempfile
import threading
import urllib.error
import urllib.request

binary = str(pathlib.Path(sys.argv[1]).resolve())
processes = []

def start(directory, *args):
    proc = subprocess.Popen(
        [binary, '--data-dir', str(directory), '--parent-stdio', *args],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        encoding='utf-8',
        errors='replace',
    )
    processes.append(proc)
    lines = queue.Queue()
    threading.Thread(target=lambda: lines.put(proc.stdout.readline()), daemon=True).start()
    try:
        line = lines.get(timeout=20)
        if not line:
            raise RuntimeError(proc.stderr.read())
        ready = json.loads(line)
        assert ready['event'] == 'ready'
        return proc, ready
    except BaseException:
        if proc.poll() is None:
            proc.kill()
            proc.wait()
        raise

def request(ready, path, body=None, token='', expected_status=200):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = 'Bearer ' + token
    req = urllib.request.Request(ready['url'] + path, data=json.dumps(body).encode() if body is not None else None, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            status, payload = response.status, response.read()
    except urllib.error.HTTPError as error:
        status, payload = error.code, error.read()
    if status != expected_status:
        raise RuntimeError(
            f'{req.get_method()} {path}: expected HTTP {expected_status}, '
            f'got {status}: {payload.decode("utf-8", errors="replace")}'
        )
    return payload

def request_data(ready, path, body=None, token=''):
    return json.loads(request(ready, path, body, token))['data']

def stop(proc):
    proc.stdin.close()
    assert proc.wait(timeout=15) == 0

def cleanup_processes():
    for proc in processes:
        if proc.poll() is None:
            proc.kill()
            proc.wait()
        for stream in (proc.stdin, proc.stdout, proc.stderr):
            stream.close()

try:
    # Stop the server before TemporaryDirectory removes its database. Windows
    # keeps open database files locked, including when a request above failed.
    with tempfile.TemporaryDirectory(prefix='questrace-smoke-') as temp, ExitStack() as cleanup:
        cleanup.callback(cleanup_processes)
        data = pathlib.Path(temp) / '中文 数据'
        busy = socket.socket()
        cleanup.callback(busy.close)
        busy.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            busy.bind(('127.0.0.1', 8080))
            busy.listen()
        except OSError:
            busy.close()  # An existing listener exercises the same fallback.
        proc, ready = start(data, '--host', '127.0.0.1')
        assert not ready['url'].endswith(':8080')
        assert b'<html' in request(ready, '/')
        request(ready, '/healthz')
        setup = {'token': ready['setup_token'], 'username': 'smoke',
                 'password': 'test-password-123', 'email': 'smoke@example.test'}
        # The production setup form requires an explicit education stage.
        # Check that requirement before exercising a valid setup.
        request(ready, '/api/v1/system/setup', setup, expected_status=400)
        request(ready, '/api/v1/system/setup',
                {**setup, 'education_stage': 'university'})
        token = request_data(ready, '/api/v1/auth/login',
                             {'username': 'smoke', 'password': 'test-password-123'})['token']
        assert request_data(ready, '/api/v1/users/me', token=token)['education_stage'] == 'university'
        question = {'subject_id': 'math_grad', 'source_type': 'manual',
                    'question_json': {'question_core': '1+1?'},
                    'mastery_status': 'unmastered', 'tags': {}}
        question_id = request_data(ready, '/api/v1/wrong-questions', question, token)['question_id']
        detail_path = f'/api/v1/wrong-questions/{question_id}'
        detail = request_data(ready, detail_path, token=token)
        assert detail['question_core'] == '1+1?'
        assert detail['subject_id'] == 'math_grad'
        assert detail['classification_status'] == 'confirmed'
        duplicate = subprocess.run([binary, '--data-dir', str(data), '--port', '0'], capture_output=True, timeout=10)
        assert duplicate.returncode != 0
        stop(proc)
        busy.close()
        archive = pathlib.Path(temp) / 'backup.zip'
        subprocess.run([binary, '--data-dir', str(data), '--backup', str(archive)], check=True, capture_output=True)
        restored = pathlib.Path(temp) / 'restored'
        subprocess.run([binary, '--data-dir', str(restored), '--restore', str(archive)], check=True, capture_output=True)
        proc, ready = start(restored, '--port', '0')
        assert 'setup_token' not in ready
        assert request_data(ready, '/api/v1/users/me', token=token)['education_stage'] == 'university'
        restored_detail = request_data(ready, detail_path, token=token)
        assert restored_detail['question_core'] == detail['question_core']
        assert restored_detail['subject_id'] == detail['subject_id']
        assert restored_detail['classification_status'] == detail['classification_status']
        stop(proc)
        print('PASS: embedded UI, occupied-port fallback, Chinese data path, exclusive lock, required education stage, setup, no-model classified question, parent EOF shutdown, backup/restore, persistent JWT/profile/data')
finally:
    cleanup_processes()
