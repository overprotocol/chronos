# Contribution Guidelines

Feel free to fork our repo and start creating PR’s after assigning yourself to an issue of interest. We are always chatting on [Discord](https://discord.com/invite/overprotocol) drop us a line there if you want to get more involved or have any questions on our implementation!

## Contribution Steps

**1. Set up Chronos following the instructions in README.md.**

**2. Fork the Chronos repo.**

Sign in to your GitHub account or create a new account if you do not have one already. Then navigate your browser to https://github.com/overprotocol/chronos/. In the upper right hand corner of the page, click “fork”. This will create a copy of the Chronos repo in your account.

**3. Create a local clone of Chronos.**

```
$ mkdir -p $GOPATH/src/github.com/overprotocol
$ cd $GOPATH/src/github.com/overprotocol
$ git clone https://github.com/overprotocol/chronos.git
$ cd $GOPATH/src/github.com/overprotocol/chronos
```

**4. Link your local clone to the fork on your GitHub repo.**

```
$ git remote add myrepo https://github.com/<your_github_user_name>/chronos.git
```

**5. Link your local clone to the Over Protocol repo so that you can easily fetch future changes to the Over Protocol repo.**

```
$ git remote add chronos https://github.com/overprotocol/chronos.git
$ git remote -v (you should see myrepo and chronos in the list of remotes)
```

**6. Find an issue to work on.**

Check out open issues at Github issue tab and pick one. Leave a comment to let the development team know that you would like to work on it. Or examine the code for areas that can be improved and leave a comment to the development team to ask if they would like you to work on it.

**7. Create a local branch with a name that clearly identifies what you will be working on.**

```
$ git checkout -b feature-in-progress-branch
```

**8. Make improvements to the code.**

Each time you work on the code be sure that you are working on the branch that you have created as opposed to your local copy of the Over Protocol repo. Keeping your changes segregated in this branch will make it easier to merge your changes into the repo later.

```
$ git checkout feature-in-progress-branch
```

**9. Test your changes.**

Changes that only affect a single file can be tested with

```
$ go test <file_you_are_working_on>
```

**10. Stage the file or files that you want to commit.**

```
$ git add --all
```

This command stages all the files that you have changed. You can add individual files by specifying the file name or names and eliminating the “-- all”.

**11. Commit the file or files.**

```
$ git commit  -m “Message to explain what the commit covers”
```

You can use the –amend flag to include previous commits that have not yet been pushed to an upstream repo to the current commit.

**12. Fetch any changes that have occurred in the Over Protocol Chronos repo since you started work.**

```
$ git fetch chronos
```

**13. Pull latest version of Chronos.**

```
$ git pull origin master
```

If there are conflicts between your edits and those made by others since you started work Git will ask you to resolve them. To find out which files have conflicts run ...

```
$ git status
```

Open those files one at a time, and you will see lines inserted by Git that identify the conflicts:

```
<<<<<< HEAD
Other developers’ version of the conflicting code
======
Your version of the conflicting code
'>>>>> Your Commit
```

The code from the Chronos repo is inserted between <<< and === while the change you have made is inserted between === and >>>>. Remove everything between <<<< and >>> and replace it with code that resolves the conflict. Repeat the process for all files listed by git status that have conflicts.

**14. Push your changes to your fork of the Chronos repo.**

Use git push to move your changes to your fork of the repo.

```
$ git push myrepo feature-in-progress-branch
```

**15. Check to be sure your fork of the Chronos repo contains your feature branch with the latest edits.**

Navigate to your fork of the repo on GitHub. On the upper left where the current branch is listed, change the branch to your feature-in-progress-branch. Open the files that you have worked on and check to make sure they include your changes.

**16. Create a pull request.**

Navigate your browser to https://github.com/overprotocol/chronos and click on the new pull request button. In the “base” box on the left, leave the default selection “base master”, the branch that you want your changes to be applied to. In the “compare” box on the right, select feature-in-progress-branch, the branch containing the changes you want to apply. You will then be asked to answer a few questions about your pull request. After you complete the questionnaire, the pull request will appear in the list of pull requests at https://github.com/overprotocol/chronos/pulls.

**17. Respond to comments by Core Contributors.**

Core Contributors may ask questions and request that you make edits. If you set notifications at the top of the page to “not watching,” you will still be notified by email whenever someone comments on the page of a pull request you have created. If you are asked to modify your pull request, repeat steps 8 through 15, then leave a comment to notify the Core Contributors that the pull request is ready for further review.

**18. If the number of commits becomes excessive, you may be asked to squash your commits.**

You can do this with an interactive rebase. Start by running the following command to determine the commit that is the base of your branch...

```
$ git merge-base feature-in-progress-branch chronos/master
```

**19. The previous command will return a commit-hash that you should use in the following command.**

```
$ git rebase -i commit-hash
```

Your text editor will open with a file that lists the commits in your branch with the word pick in front of each branch such as the following …

```
pick 	hash	do some work
pick 	hash 	fix a bug
pick 	hash 	add a feature
```

Replace the word pick with the word “squash” for every line but the first, so you end with ….

```
pick    hash	do some work
squash  hash 	fix a bug
squash  hash 	add a feature
```

Save and close the file, then a commit command will appear in the terminal that squashes the smaller commits into one. Check to be sure the commit message accurately reflects your changes and then hit enter to execute it.

**20. Update your pull request with the following command.**

```
$ git push myrepo feature-in-progress-branch -f
```

**21.  Finally, again leave a comment to the Core Contributors on the pull request to let them know that the pull request has been updated.**
