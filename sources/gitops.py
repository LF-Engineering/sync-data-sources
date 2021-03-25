#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# cython: language_level=3

import os
import datetime
import json
import subprocess
import sys
import logging

from logging import handlers
from urllib.parse import urlparse
from sys import argv


class GitOps:

    def __init__(self, url):
        self.base_path = os.getenv(
            'DA_GIT_REPOS_PATH', '~/.perceval/repositories')
        self.git_url = self.__get_processed_uri(url)
        self.uptodate = False
        self.follow_hierarchy = False
        self._cache = {}
        self.errored = False

    def __del__(self):
        pass

    @property
    def cache_path(self):
        cache_path = os.getenv('DA_GIT_CACHE_PATH', '~/.perceval/cache')
        path = os.path.expanduser(cache_path)
        if not os.path.exists(path):
            os.makedirs(path)
        return cache_path

    @property
    def cache_file_name(self):
        return 'stats.json'

    @property
    def repo_path(self):
        return self.__get_git_repo_path()

    @property
    def org_name(self):
        parser = urlparse(self.git_url)
        org_name = self._build_org_name(parser.netloc, False)
        if self.is_gitsource(parser.netloc):
            org_name = self._build_org_name(parser.path, True)
        return org_name

    @property
    def repo_name(self):
        parser = urlparse(self.git_url)
        return self._build_repo_name(parser.path, self.org_name)

    def _build_repo_name(self, path, org_name):
        sanitize_path = self.sanitize_url(path)
        if org_name in sanitize_path:
            sanitize_path = sanitize_path.replace(
                '{0}/'.format(self.org_name), '')
        if not self.follow_hierarchy:
            return sanitize_path.replace('/', '-').replace('_', '-').replace('/.', '').replace('.', '')
        return sanitize_path

    def _build_org_name(self, path, git_source):
        sanitize_path = self.sanitize_url(path)
        if not git_source:
            return sanitize_path.split('.')[1]
        return sanitize_path.split('/')[0]

    @staticmethod
    def __get_processed_uri(uri):
        removal = '.git'
        reverse_removal = removal[::-1]
        replacement = ''
        reverse_replacement = replacement[::-1]
        end = len(uri)
        start = end - 4
        if uri.endswith(removal, start, end):
            return uri[::-1].replace(reverse_removal, reverse_replacement, 1)[::-1]
        return uri

    def __get_base_path(self):
        return os.path.expanduser(self.base_path)

    def __get_cache_path(self):
        base_path = os.path.expanduser(self.cache_path)
        path = os.path.join(base_path, self.org_name)
        if not os.path.exists(path):
            os.makedirs(path)
        return path

    def __get_git_repo_path(self):
        base_path = self.__get_base_path()
        if self.follow_hierarchy:
            return os.path.join(base_path, '{0}/{1}'.format(self.org_name, self.repo_name))
        return os.path.join(base_path, '{0}-{1}'.format(self.org_name, self.repo_name))

    @staticmethod
    def _get_repo_size(start_path=None):
        total_size = 0
        if start_path:
            for dirpath, dirnames, filenames in os.walk(start_path):
                for f in filenames:
                    fp = os.path.join(dirpath, f)
                    # skip if it is symbolic link
                    if not os.path.islink(fp):
                        total_size += os.path.getsize(fp)

        return total_size

    @staticmethod
    def _get_size_format(size_bytes, factor=1024, suffix='B'):
        """
        Scale bytes to its proper byte format
        e.g:
            1253656 => '1.20MB'
            1253656678 => '1.17GB'
        """
        for unit in ['', 'K', 'M', 'G', 'T', 'P', 'E', 'Z']:
            if size_bytes < factor:
                return '{0:.2f} {1}{2}'.format(size_bytes, unit, suffix)
            size_bytes /= factor
        return '{0:.2f} Y{1}'.format(size_bytes, suffix)

    @staticmethod
    def _should_be_delete(size_unit=None):
        if size_unit:
            size, unit = size_unit.split(' ')
            if unit in ['B', 'KB']:
                return True
            elif unit == 'MB' and float(size) <= 200:
                return True
        return False

    @staticmethod
    def is_gitsource(host):
        if 'github.com' in host \
                or 'gitlab.com' in host \
                or 'bitbucket.org' in host:
            return True
        return False

    @staticmethod
    def sanitize_url(path):
        if path.startswith('/r/'):
            path = path.replace('/r/', '')
        elif path.startswith('/gerrit/'):
            path = path.replace('/gerrit/', '')
        path = path.lstrip('/')
        return path

    @staticmethod
    def sanitize_os_output(result):
        """
        Sanitize the os command output and return the readable output
        """
        sanitized_output = result.decode('UTF-8')

        return sanitized_output

    @staticmethod
    def _exec(cmd, cwd=None, env=None, ignored_error_codes=None,
              encoding='utf-8'):
        """
        Run a command.

        Execute `cmd` command in the directory set by `cwd`. Environment
        variables can be set using the `env` dictionary. The output
        data is returned as encoded bytes.

        Commands which their returning status codes are non-zero will
        be treated as failed. Error codes considered as valid can be
        ignored giving them in the `ignored_error_codes` list.

        :returns: the output of the command as encoded bytes

        :raises RepositoryError: when an error occurs running the command
        """
        if ignored_error_codes is None:
            ignored_error_codes = []

        logger.debug('Running command %s (cwd: %s, env: %s)',
                     ' '.join(cmd), cwd, str(env))

        try:
            proc = subprocess.Popen(cmd, stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE,
                                    cwd=cwd, env=env)
            (outs, errs) = proc.communicate()
        except OSError as e:
            raise RuntimeError(str(e))

        if proc.returncode != 0 and proc.returncode not in ignored_error_codes:
            err = errs.decode(encoding, errors='surrogateescape')
            cause = 'git command - %s' % err
            raise RuntimeError(cause)
        else:
            logger.debug(errs.decode(encoding, errors='surrogateescape'))

        return outs

    def _stats(self, path):
        if path and os.path.exists(path):
            cmd = ['cloc', path]
            env = {
                'LANG': 'C',
                'HOME': os.getenv('HOME', '')
            }
            return self._exec(cmd, env=env)

        return ''.encode('utf-8')

    def _pls(self, result, force=False):
        """
        Get the programing language summary
        """
        def extract_program_language_summary(value, force=False):
            stats = list()
            lan_smry_lst = value.split('\n')
            if len(lan_smry_lst) > 0 and ('SUM:' in value or force):
                for smry in lan_smry_lst[::-1]:
                    if smry.startswith('---') or len(smry) == 0:
                        continue
                    elif smry.startswith('Language'):
                        break
                    else:
                        smry_result = smry.split()
                        stats.append({
                            'language': smry_result[0].replace('SUM:', 'Total'),
                            'files': smry_result[1],
                            'blank': smry_result[2],
                            'comment': smry_result[3],
                            'code': smry_result[4]
                        })

            return stats

        return extract_program_language_summary(self.sanitize_os_output(result), force)

    def _loc(self, result, force=False):
        """
        Get the total lines of code from the default branch
        """
        def extract_lines_of_code(value, force=False):
            if len(value) > 0 and ('SUM:' in value or force):
                return int((value.split('\n')[-3]).split(' ')[-1])
            return 0

        return extract_lines_of_code(self.sanitize_os_output(result), force)

    def _clone(self):
        """
        Clone a Git repository.

        Make a bare copy of the repository stored in `uri` into `dirpath`.
        The repository would be either local or remote.

        :param uri: URI of the repository
        :param dirtpath: directory where the repository will be cloned

        :returns: a `GitRepository` class having cloned the repository

        :raises RepositoryError: when an error occurs cloning the given
            repository
        """
        cmd = ['git', 'clone', self.git_url, self.repo_path]
        env = {
            'LANG': 'C',
            'HOME': os.getenv('HOME', '')
        }

        try:
            self._exec(cmd, env=env)
            logger.debug('Git %s repository cloned into %s',
                         self.git_url, self.repo_path)
        except (RuntimeError, Exception) as cloe:
            logger.error('Git clone error %s ', str(cloe))
            self.errored = True

    def _clean(self, force=False):
        cmd = ['rm', '-rf', self.repo_path]
        env = {
            'LANG': 'C',
            'HOME': os.getenv('HOME', '')
        }

        try:
            size_bytes = self._get_repo_size(self.repo_path)
            size = self._get_size_format(size_bytes)
            if self._should_be_delete(size) or force:
                self._exec(cmd, env=env)
                logger.debug('Git %s repository clean', self.repo_path)
            else:
                logger.debug('Git %s repository clean skip', self.repo_path)
        except (RuntimeError, Exception) as cle:
            logger.error('rm error %s', str(cle))

    def _pull(self):
        os.chdir(os.path.abspath(self.repo_path))
        env = {
            'LANG': 'C',
            'HOME': os.getenv('HOME', '')
        }
        branch = None
        status = False

        try:
            cmd_auto = ['git', 'remote', 'set-head', 'origin', '--auto']
            cmd_short = ['git', 'symbolic-ref',
                         '--short', 'refs/remotes/origin/HEAD']
            self._exec(cmd_auto, env=env)
            result = self._exec(cmd_short, env=env)
            result = self.sanitize_os_output(result)
            branch = result.replace('origin/', '').strip()
            logger.debug('Git %s repository active branch is: %s',
                         self.repo_path, branch)
        except (RuntimeError, Exception) as be:
            logger.error('Git find active branch error %s', str(be))
            self.errored = True

        try:
            if branch:
                cmd = ['git', 'checkout', branch]
                self._exec(cmd, env=env)
                logger.debug('Git %s repository '
                             'checkout with following branch %s',
                             self.repo_path, branch)
        except (RuntimeError, Exception) as gce:
            logger.error('Git checkout error %s', str(gce))
            self.errored = True

        try:
            if branch:
                cmd = ['git', 'pull', 'origin', branch]
                result = self._exec(cmd, env=env)
                result = self.sanitize_os_output(result)
                if len(result) >= 18 and 'Already up to date.' in result:
                    status = True
                logger.debug('Git %s repository pull updated code',
                             self.repo_path)
            else:
                logger.debug('Git repository active branch missing')
                logger.debug('Git %s repository pull request skip ',
                             self.repo_path)
        except (RuntimeError, Exception) as pe:
            logger.error('Git pull error %s', str(pe))
            self.errored = True

        return status

    def _fetch(self):
        os.chdir(os.path.abspath(self.repo_path))

        cmd_fetch = ['git', 'fetch']
        cmd_fetch_p = ['git', 'fetch', '-p']

        env = {
            'LANG': 'C',
            'HOME': os.getenv('HOME', '')
        }

        try:
            self._exec(cmd_fetch, env=env)
            logger.debug('Git %s fetch updated code', self.repo_path)
        except (RuntimeError, Exception) as fe:
            logger.error('Git fetch purge error %s', str(fe))
            self.errored = True

        try:
            self._exec(cmd_fetch_p, env=env)
            logger.debug('Git %s fetch purge code', self.repo_path)
        except (RuntimeError, Exception) as fpe:
            logger.error('Git fetch purge error %s', str(fpe))
            self.errored = True

    def _build_empty_stats_data(self):
        stats_data = {
            self.repo_name: {
                'loc': 0,
                'pls': [],
                'timestamp': None
            }
        }
        return stats_data

    def _write_json_file(self, data, path, filename):
        try:
            path = os.path.join(path, filename)
            with open(path, 'w') as f:
                f.write(json.dumps(data, indent=4))
            f.close()
        except Exception as je:
            logger.error('cache file write error %s', str(je))
            self.errored = True
        finally:
            pass

    def _read_json_file(self, path, filename):
        error = None
        try:
            path = os.path.join(path, filename)
            with open(path, 'r') as f:
                data = f.read()
            f.close()
            return json.loads(data)
        except Exception as je:
            logger.error('cache file write error %s', str(je))
            error = True
        finally:
            if error:
                return self._build_empty_stats_data()

    def _load_cache(self):
        path = os.path.join(self.__get_cache_path(), self.cache_file_name)

        if not os.path.exists(path):
            stats_data = self._build_empty_stats_data()
            self._cache = stats_data
            self._write_json_file(data=stats_data,
                                  path=self.__get_cache_path(),
                                  filename=self.cache_file_name)
        else:
            self._cache = self._read_json_file(path=self.__get_cache_path(),
                                               filename=self.cache_file_name)

            if self.repo_name not in self._cache.keys():
                self._cache.update(self._build_empty_stats_data())
                self._write_json_file(data=self._cache,
                                      path=self.__get_cache_path(),
                                      filename=self.cache_file_name)

    def _get_cache_item(self, project_name, key):
        return self._cache[project_name][key]

    def _update_cache_item(self, project_name, key, value):
        data = self._cache.get(project_name)
        data[key] = value
        self._cache.update({project_name: data})

    def _delete_cache_item(self, project_name, key=None):
        if key:
            del self._cache[project_name][key]
        del self._cache[project_name]

    def load(self):
        if self.repo_path and not os.path.exists(self.repo_path):
            self._clone()
        else:
            self._fetch()
            self.uptodate = self._pull()

    def get_stats(self):
        loc = 0
        pls = list()

        # Get the cache loc and pls for fallback
        cache_loc = self._get_cache_item(self.repo_name, 'loc')
        cache_pls = self._get_cache_item(self.repo_name, 'pls')

        try:
            # Calculate the loc from source
            result = self._stats(self.repo_path)

            # extract new the loc and pls
            loc = self._loc(result)
            pls = self._pls(result)
            if loc == 0 and len(pls) == 0:
                loc = self._loc(result, force=True)
                pls = self._pls(result, force=True)

            logger.debug('Cache loc value %s', cache_loc)
            logger.debug('New loc value %s', loc)

            if loc == 0:
                logger.debug('LOC value set from old cache')
                # Set cache_loc value if new extracted one will be the zero
                loc = cache_loc
                pls = cache_pls
            else:
                logger.debug('Updating LOC value in cache')
                # update the cache with new value and timestamp
                self._update_cache_item(project_name=self.repo_name,
                                        key='loc',
                                        value=loc)
                self._update_cache_item(project_name=self.repo_name,
                                        key='pls',
                                        value=pls)
                utc_date = datetime.datetime.utcnow()
                if utc_date.tzinfo is None:
                    utc_date = utc_date.replace(tzinfo=datetime.timezone.utc)
                self._update_cache_item(project_name=self.repo_name,
                                        key='timestamp',
                                        value=utc_date.isoformat())
                self._write_json_file(data=self._cache,
                                      path=self.__get_cache_path(),
                                      filename=self.cache_file_name)
        except Exception as se:
            logger.error('LOC error %s', str(se))
            logger.debug('LOC value set from old cache')
            # Set cache_loc value if loc operations failed
            loc = cache_loc
            pls = cache_pls
        finally:
            logger.debug('Final LOC value %s', loc)
            return loc, pls

    def is_errored(self):
        return self.errored


logger = logging.getLogger(__name__)
git_ops = GitOps(argv[1])
git_ops._load_cache()
git_ops.load()
loc, pls = git_ops.get_stats()
if os.getenv('SKIP_CLEANUP', '') == '':
    git_ops._clean()
if git_ops.is_errored():
    sys.exit(1)
print(json.dumps({'loc': loc, 'pls': pls}))
