<script lang="ts">
  import { Plus, ArrowUp, Check } from 'lucide-svelte';
  import { api, type Question, type Answer, type User } from '../api';

  export let currentUser: User;
  export let selectedCourseId: number;
  export let currentRole: string;
  export let courseCode: string = '';

  let questions: Question[] = [];
  let selectedQuestionId: number | null = null;
  let activeQuestion: Question | null = null;
  let answers: Answer[] = [];

  // Form states
  let showAskModal = false;
  let newTitle = '';
  let newBody = '';
  let newAnswerBody = '';

  $: isTeacherOrTA = currentRole === 'Teacher' || currentRole === 'Teaching Assistant';

  $: if (currentUser && selectedCourseId) {
    loadQuestions();
  }

  async function loadQuestions() {
    if (!selectedCourseId) return;
    questions = await api.getQuestions(selectedCourseId);
    if (questions.length > 0 && selectedQuestionId === null) {
      await selectQuestion(questions[0].questionid);
    } else if (selectedQuestionId !== null) {
      await selectQuestion(selectedQuestionId);
    } else {
      activeQuestion = null;
      answers = [];
    }
  }

  async function selectQuestion(qId: number) {
    selectedQuestionId = qId;
    const res = await api.getQuestion(qId);
    activeQuestion = res.question;
    answers = res.answers;
  }

  async function handleAskQuestion() {
    if (!newTitle || !newBody) return;
    try {
      await api.createQuestion(selectedCourseId, newTitle, newBody, currentUser.userId);
      newTitle = '';
      newBody = '';
      showAskModal = false;
      await loadQuestions();
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handlePostAnswer() {
    if (!selectedQuestionId || !newAnswerBody) return;
    try {
      await api.createAnswer(selectedQuestionId, newAnswerBody, currentUser.userId);
      newAnswerBody = '';
      await selectQuestion(selectedQuestionId);
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleUpvote(answerId: number) {
    try {
      await api.upvoteAnswer(answerId);
      if (selectedQuestionId) {
        await selectQuestion(selectedQuestionId);
      }
    } catch (err: any) {
      alert(err.message);
    }
  }

  async function handleToggleOfficial(answerId: number, currentStatus: boolean) {
    try {
      await api.markOfficialAnswer(answerId, !currentStatus);
      if (selectedQuestionId) {
        await selectQuestion(selectedQuestionId);
      }
    } catch (err: any) {
      alert(err.message);
    }
  }
</script>

<div class="space-y-6">

  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-800 pb-5">
    <div>
      <h1 class="text-xl font-semibold tracking-tight text-zinc-100">Course Q&A Discussion</h1>
      <p class="text-xs text-zinc-500 mt-0.5">Threaded discussions with upvotes and verified official answers by TA/Faculty</p>
    </div>

    <button
      on:click={() => showAskModal = true}
      class="rounded-md bg-zinc-100 text-zinc-950 px-3 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors inline-flex items-center space-x-1.5"
    >
      <Plus class="w-3.5 h-3.5" />
      <span>Ask Question</span>
    </button>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">

    <!-- Questions Sidebar -->
    <div class="lg:col-span-5 space-y-2">
      <div class="text-[10px] font-mono uppercase text-zinc-500 px-1">
        Threads ({questions.length})
      </div>

      <div class="space-y-1.5">
        {#each questions as q}
          <button
            class="w-full text-left rounded-lg border p-3.5 transition-colors {selectedQuestionId === q.questionid ? 'border-zinc-700 bg-zinc-900/60' : 'border-zinc-800 bg-zinc-900/20 hover:bg-zinc-900/40'}"
            on:click={() => selectQuestion(q.questionid)}
          >
            <div class="flex items-start justify-between space-x-2">
              <h3 class="text-xs font-semibold text-zinc-100 line-clamp-1">{q.title}</h3>
              {#if q.hasofficialanswer}
                <span class="inline-flex items-center rounded border border-zinc-700 px-1.5 py-0.2 text-[9px] font-mono text-zinc-300 shrink-0">
                  SOLVED
                </span>
              {/if}
            </div>

            <p class="text-[11px] text-zinc-400 mt-1 line-clamp-2">{q.body}</p>

            <div class="mt-3 pt-2.5 border-t border-zinc-800/60 flex items-center justify-between text-[11px] text-zinc-500">
              <span>{q.firstname} {q.lastname}</span>
              <span class="font-mono">{q.answercount} answers</span>
            </div>
          </button>
        {/each}
      </div>
    </div>

    <!-- Active Thread -->
    <div class="lg:col-span-7 space-y-5">
      {#if activeQuestion}
        
        <!-- Question Box -->
        <div class="rounded-lg border border-zinc-800 bg-zinc-900/30 p-5 space-y-3">
          <div class="flex items-center space-x-2 text-[10px] font-mono text-zinc-500">
            <span>QUESTION #{activeQuestion.questionid}</span>
            <span>•</span>
            <span>COURSE {courseCode || '—'}</span>
          </div>

          <h2 class="text-base font-semibold text-zinc-100">{activeQuestion.title}</h2>
          <p class="text-xs text-zinc-300 whitespace-pre-wrap leading-relaxed">{activeQuestion.body}</p>

          <div class="pt-3 border-t border-zinc-800/60 text-[11px] text-zinc-500">
            Asked by <span class="text-zinc-300 font-medium">{activeQuestion.firstname} {activeQuestion.lastname}</span>
          </div>
        </div>

        <!-- Answers List -->
        <div class="space-y-3">
          <div class="text-[10px] font-mono uppercase text-zinc-500 px-1">
            Answers ({answers.length})
          </div>

          {#each answers as ans}
            <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 p-4 space-y-2.5">
              
              <div class="flex items-center justify-between">
                <div class="flex items-center space-x-2 text-xs">
                  <span class="font-medium text-zinc-200">{ans.firstname} {ans.lastname}</span>
                  {#if ans.userrole?.String}
                    <span class="font-mono text-[10px] text-zinc-500 border border-zinc-800 px-1 py-0.2 rounded">
                      {ans.userrole.String}
                    </span>
                  {/if}
                </div>

                {#if ans.isofficialanswer?.Bool}
                  <span class="inline-flex items-center space-x-1 rounded border border-zinc-700 bg-zinc-800/40 px-1.5 py-0.5 text-[10px] font-mono text-zinc-200">
                    <Check class="w-3 h-3 text-zinc-300" />
                    <span>VERIFIED ANSWER</span>
                  </span>
                {/if}
              </div>

              <p class="text-xs text-zinc-300 leading-relaxed whitespace-pre-wrap">{ans.body}</p>

              <!-- Answer Controls -->
              <div class="pt-2.5 border-t border-zinc-800/60 flex items-center justify-between">
                <button
                  on:click={() => handleUpvote(ans.answerid)}
                  class="inline-flex items-center space-x-1 rounded border px-2 py-1 text-xs transition-colors font-mono {ans.hasupvoted
                    ? 'border-zinc-500 bg-zinc-800 text-zinc-100'
                    : 'border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900'}"
                >
                  <ArrowUp class="w-3.5 h-3.5" />
                  <span>{ans.upvotes?.Int32 || 0}</span>
                </button>

                {#if isTeacherOrTA}
                  <button
                    on:click={() => handleToggleOfficial(ans.answerid, !!ans.isofficialanswer?.Bool)}
                    class="text-[11px] font-medium px-2 py-0.5 rounded border border-zinc-800 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors"
                  >
                    {ans.isofficialanswer?.Bool ? 'Revoke official' : 'Mark as official'}
                  </button>
                {/if}
              </div>

            </div>
          {/each}

          <!-- Write Answer -->
          <div class="rounded-lg border border-zinc-800 bg-zinc-900/20 p-4 space-y-3">
            <label for="answer-textarea" class="block text-xs font-medium text-zinc-400">Post Answer</label>
            <textarea
              id="answer-textarea"
              bind:value={newAnswerBody}
              rows="3"
              placeholder="Write your explanation or code solution..."
              class="w-full rounded-md border border-zinc-800 bg-zinc-950 p-2.5 text-xs text-zinc-100 placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-400"
            ></textarea>
            <div class="flex justify-end">
              <button
                on:click={handlePostAnswer}
                class="rounded-md bg-zinc-100 text-zinc-950 px-3 py-1.5 text-xs font-medium hover:bg-zinc-200 transition-colors"
              >
                Submit Answer
              </button>
            </div>
          </div>
        </div>

      {:else}
        <div class="rounded-lg border border-zinc-800 p-8 text-center text-xs text-zinc-500">
          Select a thread from the list.
        </div>
      {/if}
    </div>

  </div>

  <!-- Ask Question Modal -->
  {#if showAskModal}
    <div class="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4">
      <div class="bg-zinc-950 border border-zinc-800 rounded-lg max-w-md w-full p-5 space-y-4 shadow-xl">
        <h3 class="text-sm font-semibold text-zinc-100">Ask a Question</h3>
        
        <div>
          <label for="question-title-input" class="block text-xs font-medium text-zinc-400 mb-1">Title</label>
          <input
            id="question-title-input"
            bind:value={newTitle}
            placeholder="e.g. How does BCNF handle multi-attribute determinants?"
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-100 placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-400"
          />
        </div>

        <div>
          <label for="question-body-input" class="block text-xs font-medium text-zinc-400 mb-1">Details</label>
          <textarea
            id="question-body-input"
            bind:value={newBody}
            rows="4"
            placeholder="Provide context and code snippet if applicable..."
            class="w-full rounded-md border border-zinc-800 bg-zinc-900 p-2.5 text-xs text-zinc-100 placeholder:text-zinc-600 focus:outline-none focus:ring-1 focus:ring-zinc-400"
          ></textarea>
        </div>

        <div class="flex justify-end space-x-2 pt-2 border-t border-zinc-800">
          <button on:click={() => showAskModal = false} class="px-3 py-1.5 text-xs rounded-md border border-zinc-800 text-zinc-400 hover:text-zinc-100">Cancel</button>
          <button on:click={handleAskQuestion} class="px-3 py-1.5 text-xs rounded-md bg-zinc-100 text-zinc-950 font-medium hover:bg-zinc-200">Post</button>
        </div>
      </div>
    </div>
  {/if}

</div>
