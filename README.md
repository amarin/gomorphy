# gomorphy
Golang based PyMorphy2 analog.

## How to use

1. Took fresh index from opencorpora.org using opencorpora_update. It will load last index, rebuild and save in under the .data 
2. Check tags are successfully extracted using opencorpora_test utility.
3. Make your own application 
4. Implement compiled index loading using opencorpora loader and its LoadIndex method. Use opencorpora_test source code as implementation example
5. Implement index search using loaded index fetchString method. Use opencorpora_test/main.go/processSearch source code as implementation example


