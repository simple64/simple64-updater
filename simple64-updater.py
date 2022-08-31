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


def update_simple64(root2: tk.Tk, var2: tk.StringVar) -> None:
    var2.set("Determining latest release")
    resp = requests.get(
        "https://api.github.com/repos/simple64/simple64/releases/latest"
    )
    if resp.status_code != 200:
        root2.quit()
        return
    for item in resp.json()["assets"]:
        if sys.platform.startswith("win") and "simple64-win64" in item["name"]:
            simple64_url = item["browser_download_url"]
        elif sys.platform.startswith("lin") and "simple64-linux64" in item["name"]:
            simple64_url = item["browser_download_url"]

    var2.set("Downloading latest release")
    resp = requests.get(simple64_url, allow_redirects=True)
    if resp.status_code != 200:
        root2.quit()
        return

    with tempfile.TemporaryDirectory() as tempdir:
        filename = os.path.join(tempdir, "simple64.zip")
        with open(filename, "wb") as localfile:
            localfile.write(resp.content)

        var2.set("Extracting release")
        with zipfile.ZipFile(filename, "r") as zf:
            for info in zf.infolist():
                zf.extract(info.filename, path=tempdir)
                out_path = os.path.join(tempdir, info.filename)
                perm = info.external_attr >> 16
                try:
                    os.chmod(out_path, perm)
                except OSError:
                    pass

        var2.set("Moving files into place")
        extract_path = os.path.join(tempdir, "simple64")
        shutil.copytree(extract_path, sys.argv[1], dirs_exist_ok=True)

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
    root.title("simple64-updater")
    var = tk.StringVar()
    var.set("Initializing")
    w = tk.Label(root, textvariable=var)
    w.pack(fill="none", expand=True)

    x = threading.Thread(target=update_simple64, args=(root, var))
    root.after(3000, start_thread, x)
    root.mainloop()
    x.join()

    subprocess.Popen(
        [os.path.join(sys.argv[1], "simple64-gui")],
        env=my_env,
        start_new_session=True,
        close_fds=True,
    )


if __name__ == "__main__":
    main()
