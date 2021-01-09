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


def update_m64p(root2: tk.Tk, var2: tk.StringVar) -> None:
    var2.set("Determining latest release")
    resp = requests.get(
        'https://api.github.com/repos/loganmc10/m64p/releases/latest')
    if resp.status_code != 200:
        root2.quit()
        return
    for item in resp.json()['assets']:
        if sys.platform.startswith('win') and 'm64p-win64' in item['name']:
            m64p_url = item['browser_download_url']
        elif sys.platform.startswith('lin') and 'm64p-linux64' in item['name']:
            m64p_url = item['browser_download_url']

    var2.set("Downloading latest release")
    resp = requests.get(m64p_url, allow_redirects=True)

    with tempfile.TemporaryDirectory() as tempdir:
        filename = os.path.join(tempdir, 'm64p.zip')
        with open(filename, 'wb') as localfile:
            localfile.write(resp.content)

        var2.set("Extracting release")
        with zipfile.ZipFile(filename, 'r') as zf:
            for info in zf.infolist():
                zf.extract(info.filename, path=tempdir)
                out_path = os.path.join(tempdir, info.filename)
                perm = (info.external_attr >> 16)
                try:
                    os.chmod(out_path, perm)
                except:
                    pass

        var2.set("Moving files into place")
        extract_path = os.path.join(tempdir, 'mupen64plus')
        files = os.listdir(extract_path)
        for f in files:
            shutil.copy2(os.path.join(extract_path, f),
                         os.path.join(sys.argv[1], f))

    var2.set("Cleaning up")
    root2.quit()


def start_thread(x2: threading.Thread) -> None:
    x2.start()


def main() -> None:
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

    x = threading.Thread(target=update_m64p, args=(root, var))
    root.after(3000, start_thread, x)
    root.mainloop()
    x.join()

    subprocess.Popen([os.path.join(sys.argv[1], 'mupen64plus-gui')],
                     env=my_env, start_new_session=True, close_fds=True)


if __name__ == '__main__':
    main()
