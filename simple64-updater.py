#!/usr/bin/python3

import threading
import subprocess  # nosec
import requests
import tempfile
import sys
import os
import zipfile
import shutil
import tkinter as tk


def clean_dir(install_path: str) -> None:
    # loop through the directory twice, one to clean up files, then again to remove empty dirs
    for root, _, files in os.walk(install_path):
        for name in files:
            path = os.path.join(root, name)
            if path.endswith(".dll") or path.endswith(".exe"):
                try:
                    os.remove(path)
                except OSError:
                    pass

    for root, dirs, _ in os.walk(install_path, topdown=False):  # remove empty dirs
        for name in dirs:
            try:
                os.rmdir(os.path.join(root, name))
            except OSError:
                pass


def update_simple64(root2: tk.Tk, var2: tk.StringVar) -> None:
    var2.set("Determining latest release")
    resp = requests.get(
        "https://api.github.com/repos/simple64/simple64/releases/latest", timeout=30
    )
    if resp.status_code != 200:
        root2.quit()
        return
    simple64_url = ""
    for item in resp.json()["assets"]:
        if sys.platform.startswith("win") and "simple64-win64" in item["name"]:
            simple64_url = item["browser_download_url"]

    if not simple64_url:  # quit early if URL couldn't be determined
        root2.quit()
        return

    var2.set("Downloading latest release")
    resp = requests.get(simple64_url, allow_redirects=True, timeout=30)
    if resp.status_code != 200:
        root2.quit()
        return

    clean_dir(sys.argv[1])  # clean up current directory

    with tempfile.TemporaryDirectory() as tempdir:
        filename = os.path.join(tempdir, "simple64.zip")
        with open(filename, "wb") as localfile:
            localfile.write(resp.content)  # write zip into temp dir

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

    subprocess.Popen(  # nosec
        [os.path.join(sys.argv[1], "simple64-gui")],
        env=my_env,
        start_new_session=True,
        close_fds=True,
    )


if __name__ == "__main__":
    main()
