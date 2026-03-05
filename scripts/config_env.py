import sys
import os

def set_deb_control(control_path, version, arch):
    file_name = f"{control_path}/control"
    os.makedirs(os.path.dirname(file_name), exist_ok=True)  # ensure directory exists

    template = f"""Package: easee
Section: non-free/misc
Version: {version}
Architecture: {arch}
Priority: optional
Replaces: easee
Maintainer: Futurehome AS <dev@futurehome.no>
Description: Futurehome Easee EV charger adapter
"""

    try:
        with open(file_name, "w") as f:
            f.write(template)
            print(f"DEBIAN control file written to {file_name}")
    except Exception as e:
        print(f"Error writing control file: {e}")
        sys.exit(1)

if __name__ == "__main__":
   control_path = sys.argv[1]
   version = sys.argv[2]
   arch = sys.argv[3]
   set_deb_control(control_path, version, arch)
