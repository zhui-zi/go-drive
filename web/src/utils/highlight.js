
import hljs from 'highlight.js/lib/core'

import actionscript from 'highlight.js/lib/languages/actionscript'
import angelscript from 'highlight.js/lib/languages/angelscript'
import apache from 'highlight.js/lib/languages/apache'
import c$like from 'highlight.js/lib/languages/c-like'
import cpp from 'highlight.js/lib/languages/cpp'
import arduino from 'highlight.js/lib/languages/arduino'
import armasm from 'highlight.js/lib/languages/armasm'
import xml from 'highlight.js/lib/languages/xml'
import asciidoc from 'highlight.js/lib/languages/asciidoc'
import aspectj from 'highlight.js/lib/languages/aspectj'
import autohotkey from 'highlight.js/lib/languages/autohotkey'
import autoit from 'highlight.js/lib/languages/autoit'
import awk from 'highlight.js/lib/languages/awk'
import bash from 'highlight.js/lib/languages/bash'
import bnf from 'highlight.js/lib/languages/bnf'
import c from 'highlight.js/lib/languages/c'
import clojure from 'highlight.js/lib/languages/clojure'
import clojure$repl from 'highlight.js/lib/languages/clojure-repl'
import cmake from 'highlight.js/lib/languages/cmake'
import coffeescript from 'highlight.js/lib/languages/coffeescript'
import csharp from 'highlight.js/lib/languages/csharp'
import css from 'highlight.js/lib/languages/css'
import markdown from 'highlight.js/lib/languages/markdown'
import dart from 'highlight.js/lib/languages/dart'
import diff from 'highlight.js/lib/languages/diff'
import dockerfile from 'highlight.js/lib/languages/dockerfile'
import dos from 'highlight.js/lib/languages/dos'
import elixir from 'highlight.js/lib/languages/elixir'
import ruby from 'highlight.js/lib/languages/ruby'
import erlang$repl from 'highlight.js/lib/languages/erlang-repl'
import erlang from 'highlight.js/lib/languages/erlang'
import excel from 'highlight.js/lib/languages/excel'
import go from 'highlight.js/lib/languages/go'
import gradle from 'highlight.js/lib/languages/gradle'
import groovy from 'highlight.js/lib/languages/groovy'
import haskell from 'highlight.js/lib/languages/haskell'
import http from 'highlight.js/lib/languages/http'
import ini from 'highlight.js/lib/languages/ini'
import java from 'highlight.js/lib/languages/java'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import julia from 'highlight.js/lib/languages/julia'
import julia$repl from 'highlight.js/lib/languages/julia-repl'
import kotlin from 'highlight.js/lib/languages/kotlin'
import latex from 'highlight.js/lib/languages/latex'
import less from 'highlight.js/lib/languages/less'
import lisp from 'highlight.js/lib/languages/lisp'
import llvm from 'highlight.js/lib/languages/llvm'
import lua from 'highlight.js/lib/languages/lua'
import makefile from 'highlight.js/lib/languages/makefile'
import matlab from 'highlight.js/lib/languages/matlab'
import perl from 'highlight.js/lib/languages/perl'
import nginx from 'highlight.js/lib/languages/nginx'
import objectivec from 'highlight.js/lib/languages/objectivec'
import ocaml from 'highlight.js/lib/languages/ocaml'
import pgsql from 'highlight.js/lib/languages/pgsql'
import php from 'highlight.js/lib/languages/php'
import plaintext from 'highlight.js/lib/languages/plaintext'
import powershell from 'highlight.js/lib/languages/powershell'
import properties from 'highlight.js/lib/languages/properties'
import protobuf from 'highlight.js/lib/languages/protobuf'
import python from 'highlight.js/lib/languages/python'
import qml from 'highlight.js/lib/languages/qml'
import r from 'highlight.js/lib/languages/r'
import rust from 'highlight.js/lib/languages/rust'
import scala from 'highlight.js/lib/languages/scala'
import scheme from 'highlight.js/lib/languages/scheme'
import scss from 'highlight.js/lib/languages/scss'
import shell from 'highlight.js/lib/languages/shell'
import smali from 'highlight.js/lib/languages/smali'
import sql from 'highlight.js/lib/languages/sql'
import stylus from 'highlight.js/lib/languages/stylus'
import swift from 'highlight.js/lib/languages/swift'
import yaml from 'highlight.js/lib/languages/yaml'
import typescript from 'highlight.js/lib/languages/typescript'
import vbnet from 'highlight.js/lib/languages/vbnet'
import vbscript from 'highlight.js/lib/languages/vbscript'
import vbscript$html from 'highlight.js/lib/languages/vbscript-html'
import vim from 'highlight.js/lib/languages/vim'
import x86asm from 'highlight.js/lib/languages/x86asm'

const languages = {
  actionscript, angelscript, apache, c$like, cpp, arduino, armasm, xml, asciidoc, aspectj, autohotkey, autoit,
  awk, bash, bnf, c, clojure, clojure$repl, cmake, coffeescript, csharp, css, markdown, dart, diff, dockerfile, dos,
  elixir, ruby, erlang$repl, erlang, excel, go, gradle, groovy, haskell, http, ini, java, javascript, json,
  julia, julia$repl, kotlin, latex, less, lisp, llvm, lua, makefile, matlab, perl, nginx, objectivec, ocaml, pgsql,
  php, plaintext, powershell, properties, protobuf, python, qml, r, rust, scala, scheme, scss, shell, smali, sql,
  stylus, swift, yaml, typescript, vbnet, vbscript, vbscript$html, vim, x86asm
}

Object.keys(languages).forEach(key => {
  const lang = languages[key]
  key = key.replace(/^\$+/, '').replace('$', '-')
  hljs.registerLanguage(key, lang)
})

export default hljs