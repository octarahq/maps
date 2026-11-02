package com.octara.maps.navigation

import androidx.compose.runtime.Composable
import androidx.compose.runtime.ProvidableCompositionLocal
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue

sealed interface Screen {
    data object Map : Screen
    data object Search : Screen
    data class StopInfo(val stopId: String) : Screen
}

class Router(initialScreen: Screen = Screen.Map) {
    var currentScreen by mutableStateOf(initialScreen)
        private set

    private val backStack = mutableListOf(initialScreen)

    val canGoBack: Boolean
        get() = backStack.size > 1

    fun navigateTo(screen: Screen, popCurrent: Boolean = false) {
        if (popCurrent && backStack.size > 1) {
            backStack.removeAt(backStack.size - 1)
        }
        if (currentScreen != screen) {
            backStack.add(screen)
            currentScreen = screen
        }
    }

    fun goBack(): Boolean {
        if (canGoBack) {
            backStack.removeAt(backStack.size - 1)
            currentScreen = backStack.last()
            return true
        }
        return false
    }

    fun popToRoot() {
        if (backStack.size > 1) {
            val root = backStack.first()
            backStack.clear()
            backStack.add(root)
            currentScreen = root
        }
    }
}

val LocalRouter: ProvidableCompositionLocal<Router> = compositionLocalOf {
    error("no router provided")
}

@Composable
fun rememberRouter(initialScreen: Screen = Screen.Map): Router {
    return remember { Router(initialScreen) }
}
