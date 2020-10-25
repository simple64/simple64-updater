#!/usr/bin/python3

import requests
import tempfile
import sys
import os
import zipfile
import shutil

if len(sys.argv) < 2:
    print("no argument!")
    sys.exit(1)

resp = requests.get('https://api.github.com/repos/loganmc10/m64p/releases/latest')
if resp.status_code != 200:
    # This means something went wrong.
    raise ApiError('GET /tasks/ {}'.format(resp.status_code))
latest = str(resp.json()['id'])
resp = requests.get('https://api.github.com/repos/loganmc10/m64p/releases/' + latest + '/assets')
if resp.status_code != 200:
    # This means something went wrong.
    raise ApiError('GET /tasks/ {}'.format(resp.status_code))
for item in resp.json():
    if sys.platform.startswith('win') and 'win' in item['browser_download_url']:
        m64p_url = item['browser_download_url']
    elif sys.platform.startswith('lin') and 'lin' in item['browser_download_url']:
        m64p_url = item['browser_download_url']

resp = requests.get(m64p_url, allow_redirects=True)
tempdir = tempfile.mkdtemp()
filename = os.path.join(tempdir, 'm64p.zip')
open(filename, 'wb').write(resp.content)

with zipfile.ZipFile(filename, 'r') as zip_ref:
    zip_ref.extractall(tempdir)

extract_path = os.path.join(tempdir, 'mupen64plus')
files = os.listdir(extract_path)
for f in files:
    shutil.move(os.path.join(extract_path, f), os.path.join(sys.argv[1], f))

shutil.rmtree(tempdir)
sys.exit(0)
