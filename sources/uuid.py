#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#cython: language_level=3

from hashlib import sha1
from unicodedata import normalize, category
from sys import argv

def to_unicode(x, unaccent=False):
    """Convert a string to unicode"""
    s = str(x)
    if x == '<nil>':
        s = 'None'
    if unaccent:
        cs = [c for c in normalize('NFD', s)
              if category(c) != 'Mn']
        s = ''.join(cs)
    return s

def uuida(args):
    def check_value(v):
        if not isinstance(v, str):
            raise ValueError("%s value is not a string instance" % str(v))
        elif not v:
            raise ValueError("value cannot be None or empty")
        else:
            return v
    s = ':'.join(map(check_value, args))
    # print(s.encode('utf-8', errors="surrogateescape"))
    sha = sha1(s.encode('utf-8', errors='surrogateescape'))
    uuid_sha = sha.hexdigest()
    return uuid_sha

def uuid(source, email=None, name=None, username=None):
    if source is None:
        raise ValueError("source cannot be None")
    if source == '':
        raise ValueError("source cannot be an empty string")
    if not (email or name or username):
        raise ValueError("identity data cannot be None or empty")
    s = ':'.join((to_unicode(source),
                  to_unicode(email),
                  to_unicode(name, unaccent=True),
                  to_unicode(username))).lower()
    # print(s.encode('UTF-8', errors="surrogateescape"))
    sha = sha1(s.encode('UTF-8', errors="surrogateescape"))
    uuid_ = sha.hexdigest()
    return uuid_

if argv[1] == 'a':
    print(uuida(argv[2:]))
else:
    print(uuid(argv[2], argv[3], argv[4], argv[5]))
