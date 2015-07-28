# gitlab-git-http-server

This is a proof of concept for unloading Git HTTP traffic from the
GitLab Rails app (Unicorn) to a separate daemon. All authentication
and authorization logic is still handled by the GitLab Rails app.

Architecture: Git client -> NGINX -> gitlab-git-http-server (makes
auth request to GitLab Rails app) -> git-upload-pack

There are two patches in the repo that show what would need to
change in GitLab / NGINX to make this work.

## Example request flow

- start POST repo.git/git-receive-pack to NGINX
- ..start POST repo.git/git-receive-pack to gitlab-git-http-server
- ....start POST repo.git/git-receive-pack to Unicorn for auth
- ....end POST to Unicorn for auth
- ....start git-receive-pack process from gitlab-git-http-server
- ......start POST /api/v3/internal/allowed to Unicorn from Git hook (check protected branches)
- ......end POST to Unicorn from Git hook
- ....end git-receive-pack process
- ..end POST to gitlab-git-http-server
- end POST to NGINX

## Try it out

You can try out the Git server without authentication as follows:

```
# Start a fake auth backend that allows everything/everybody
go run support/say-yes.go &
# Start gitlab-git-http-server
go run main.go /path/to/git-repos
```

Now if you have a Git repository in `/path/to/git-repos/my-repo.git`,
you can push to and pull from it at the URL
`http://localhost:8181/my-repo.git`.

## License

This code is distributed under the MIT license, see the LICENSE file.