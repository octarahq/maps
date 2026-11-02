package com.octara.maps.ui.screens.map

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.BottomSheetScaffold
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.rememberBottomSheetScaffoldState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.navigation.Screen
import com.octara.maps.ui.components.AppSearchBar
import com.octara.maps.ui.theme.AppDimensions

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransitMapScreen(
    modifier: Modifier = Modifier,
    onSearchClick: (() -> Unit)? = null
) {
    val router = LocalRouter.current
    val scaffoldState = rememberBottomSheetScaffoldState()

    BottomSheetScaffold(
        scaffoldState = scaffoldState,
        sheetShape = AppDimensions.BottomSheetShape,
        sheetPeekHeight = AppDimensions.BottomSheetPeekHeight,
        sheetShadowElevation = AppDimensions.BottomSheetElevation,
        sheetContent = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp)
            ) {
                AppSearchBar(
                    active = false,
                    onClick = onSearchClick ?: { router.navigateTo(Screen.Search) }
                )
            }
        },
        modifier = modifier.fillMaxSize()
    ) {
        TransitMap(modifier = Modifier.fillMaxSize())
    }
}
