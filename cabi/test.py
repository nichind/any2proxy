import ctypes
import json
import sys


def main(path):
    lib = ctypes.CDLL(path)
    lib.a2p_version.restype = ctypes.c_void_p
    lib.a2p_new.restype = ctypes.c_uint64
    lib.a2p_call.argtypes = [ctypes.c_uint64, ctypes.c_char_p, ctypes.c_char_p]
    lib.a2p_call.restype = ctypes.c_void_p
    lib.a2p_free.argtypes = [ctypes.c_void_p]
    lib.a2p_close.argtypes = [ctypes.c_uint64]

    version = lib.a2p_version()
    print(ctypes.string_at(version).decode())
    lib.a2p_free(version)

    handle = lib.a2p_new()

    def call(method, params=None):
        raw = lib.a2p_call(handle, method.encode(), json.dumps(params or {}).encode())
        data = json.loads(ctypes.string_at(raw).decode())
        lib.a2p_free(raw)
        return data

    print(call("version"))
    result = call(
        "compile",
        {
            "body": "vless://u@example.com:443#one\n",
            "format": "plain",
            "target": "mihomo",
            "port": 1080,
        },
    )
    print(result["result"]["config"].splitlines()[0])
    lib.a2p_close(handle)


if __name__ == "__main__":
    main(sys.argv[1])
