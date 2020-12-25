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


def update_m64p():
    var.set("Determining latest release")
    resp = requests.get('https://api.github.com/repos/loganmc10/m64p/releases/latest')
    if resp.status_code != 200:
        return
    for item in resp.json()['assets']:
        if sys.platform.startswith('win') and 'm64p-win64' in item['name']:
            m64p_url = item['browser_download_url']
        elif sys.platform.startswith('lin') and 'm64p-linux64' in item['name']:
            m64p_url = item['browser_download_url']

    var.set("Downloading latest release")
    resp = requests.get(m64p_url, allow_redirects=True)

    with tempfile.TemporaryDirectory() as tempdir:
        filename = os.path.join(tempdir, 'm64p.zip')
        with open(filename, 'wb') as localfile:
            localfile.write(resp.content)

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

    root.quit()


def start_thread():
    x = threading.Thread(target=update_m64p)
    x.start()


if len(sys.argv) < 2:
    print("no argument!")
    sys.exit(1)

my_env = os.environ.copy()

root = tk.Tk()
root.geometry("400x200")
root.title("m64p-updater")
var = tk.StringVar()
var.set("Initializing")
w = tk.Label(root, textvariable=var)
w.pack(fill="none", expand=True)

root.after(3000, start_thread)
root.mainloop()

subprocess.Popen([os.path.join(sys.argv[1], 'mupen64plus-gui')], env=my_env, start_new_session=True, close_fds=True)
