---
layout: post
title: Garbage Collection
date: 2023-01-24 8:00am
categories: [Memory Management]
tags: [gc, garbage-collector, memory, memory-management]
---
<h1> Garbage Collection </h1>

<h3 dir="rtl" lang="ar">ايه هي الـ Garbage Collection ؟</h3>
<p dir="rtl" lang="ar">
الـ Garbage Collection هي عمليه إدارة اوتوماتيكية للذاكرة (automatic memory management)، وبتستخدمها معظم لغات البرمجة الحديثة زي Python, Java, Golang..etc عشان تسهل عمليه تطوير البرامج من غير منهتم بتفاصيل الـ low-level memory.
</p>

<br>

<h3 dir="rtl" lang="ar"> ايه فايدة الـ Garbage Collector</h3>

<p dir="rtl" lang="ar">
فـ لغات زي الـ ++C/C مفيهاش Garbage Collector فالمبرمج بيكون عليه مسؤولية الـ allocation و الـ deallocation للـ memory؛ لكن ايه اللي هيحصل لو المبرمج نسا انه يعمل deallocate للـ variables اللي مبقتش مستخدمة فـ البرنامج؟ او عمل deallocate بالغلط لـ variable لسا بيستخدم فالبرناج؟

<h5 dir="rtl" lang="ar">في حال نسيان تظيف الذاكرة (Forgetting to free the memory):</h5>

<p dir="rtl" lang="ar">
لما ميتمش تنظيف الـ memory بعد استخدامها دا بيأدي لـ <a href="https://en.wikipedia.org/wiki/Memory_leak">memory leaks</a>، يعني تعمل allocate وتنسا تـ deallocate ودا طبعا بيقلل اداء الكومبيوتر لان مساحة الـ memory بتقل وطبعا يخلي البرنامج vulnerable.
</p>

<h5 dir="rtl" lang="ar"> في حال التنظيف المبكر للذاكرة (Freeing the memory too soon):</h5>

<p dir="rtl" lang="ar">
لما يتم تنظيف الـ memory وهي مازالت بتستخدم ف البرنامج، البرنامج هيـ crash لما يحاول يوصل لـ value مش موجودة في الـ memory (الـ value اللي تم تنظيفها بالغلط)، أو يكون في variables بتُشير (أو بتـ refers) للـ memory اللي تم تنظيفها فيكون في <a href="https://en.wikipedia.org/wiki/Dangling_pointer">dangling pointers</a> ودي pointers بتُشير لـ invalid memory location.
</p>
<p dir="rtl" lang="ar">
المشاكل دي حلتها الـ Garbage Collector يعني  المبرمج مش هيشغل بالو بتنظيف الذاكرة بعد الـ allocation لأنها بدير كل العمليات دي بشكل اوتوماتيكي (automatic memory management).
</p>

<br>

<h3 dir="rtl" lang="ar">تكلفة الـ Garbage Collection</h3>

<p dir="rtl" lang="ar">
الـ Garbage Collection بيحتاج مساحة إضافية تتسع لأي عدد من الـ pointers في البرنامج لأنها مسؤلة عن تتبع عدد الـ reference لكل object، لما الـ object يكون zero references معندهوش اي reference، يعني كدا الـ object غير مستخدم في البرنامج، وبالتالي هيتم مسحه أو freeing.
</p>
<p dir="rtl" lang="ar">
أثناء تحقق الـ Garbage Collector من أن الـ objects مستخدمة ولا لا في البرنامج بتوقف تنفيد البرنامج عبل ميتم مسح (delete) الـ objects الغير مستخدمة، ودا سبب يخلي الـ performance أسرع في اللغات اللي مبتستخدمش Garbage Collection.
</p>
