#!/usr/bin/python3

import threading
import subprocess
import requests
import tempfile
import sys
import os
import zipfile
import shutil
import tkinter as tk

if len(sys.argv) < 2:
    print("no argument!")
    sys.exit(1)

root = tk.Tk()
root.geometry("400x200")
root.title("m64p-updater")
var = tk.StringVar()
var.set("Initializing")
w = tk.Label(root, textvariable=var)
w.pack(fill="none", expand=True)

def update_m64p():
    var.set("Determining latest release")
    resp = requests.get('https://api.github.com/repos/loganmc10/m64p/releases/latest')
    if resp.status_code != 200:
        raise ApiError('GET /tasks/ {}'.format(resp.status_code))
    latest = str(resp.json()['id'])
    resp = requests.get('https://api.github.com/repos/loganmc10/m64p/releases/' + latest + '/assets')
    if resp.status_code != 200:
       raise ApiError('GET /tasks/ {}'.format(resp.status_code))
    for item in resp.json():
        if sys.platform.startswith('win') and 'win' in item['browser_download_url']:
            m64p_url = item['browser_download_url']
        elif sys.platform.startswith('lin') and 'lin' in item['browser_download_url']:
            m64p_url = item['browser_download_url']

    var.set("Downloading latest release")
    resp = requests.get(m64p_url, allow_redirects=True)
    tempdir = tempfile.mkdtemp()
    filename = os.path.join(tempdir, 'm64p.zip')
    open(filename, 'wb').write(resp.content)

    var.set("Extracting release")
    with zipfile.ZipFile(filename, 'r') as zf:
        for info in zf.infolist():
            zf.extract( info.filename, path=tempdir )
            out_path = os.path.join( tempdir, info.filename )
            perm = (info.external_attr >> 16)
            os.chmod( out_path, perm )

    extract_path = os.path.join(tempdir, 'mupen64plus')
    files = os.listdir(extract_path)
    for f in files:
        shutil.move(os.path.join(extract_path, f), os.path.join(sys.argv[1], f))

    var.set("Cleaning up")
    shutil.rmtree(tempdir)

    root.quit()

def start_thread():
    x = threading.Thread(target=update_m64p)
    x.start()

root.after(3000, start_thread)
root.mainloop()

subprocess.Popen([os.path.join(sys.argv[1], 'mupen64plus-gui')])
