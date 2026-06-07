REPO_NAME := 'website'
GITHUB_REPO := '0x61nas.github.io'

setup-remotes:
    git remote add github git@github.com:0x61nas/{{GITHUB_REPO}}.git
    git remote add gitlab git@gitlab.com:anelgarhy/{{REPO_NAME}}.git
    git remote add codeberg ssh://git@codeberg.org/0x61nas/{{REPO_NAME}}.git
    git remote add disroot ssh://git@git.disroot.org/anas/{{REPO_NAME}}.git
    git remote add tangled git@tangled.org:anas.tngl.sh/{{REPO_NAME}}
    git remote add codefloe ssh://git@codefloe.com/anas/{{REPO_NAME}}.git
    git remote add gitea git@gitea.com:0x61nas/{{REPO_NAME}}.git
    git remote add gitgud git@ssh.gitgud.io:anelgarhy/{{REPO_NAME}}.git

# Push the code to all remotes
push FLAGS="-u" BRANSH="aurora":
    git push {{FLAGS}} github {{BRANSH}}
    git push {{FLAGS}} gitlab {{BRANSH}}
    git push {{FLAGS}} codeberg {{BRANSH}}
    git push {{FLAGS}} disroot {{BRANSH}}
    git push {{FLAGS}} tangled {{BRANSH}}
    git push {{FLAGS}} codefloe {{BRANSH}}
    git push {{FLAGS}} gitea {{BRANSH}} 
    git push {{FLAGS}} gitgud {{BRANSH}} 

# Push the git tags to all remotes
pusht: push
    git push --tags github
    git push --tags gitlab
    git push --tags codeberg
    git push --tags disroot
    git push --tags tangled
    git push --tags codefloe
    git push --tags gitea
    git push --tags gitgud

clean:
    git clean -ffdx
